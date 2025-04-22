package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
	"github.com/siteworxpro/top-wallpaper/resize"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type redditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After     string `json:"after"`
		Dist      int    `json:"dist"`
		ModHash   string `json:"modhash"`
		GeoFilter string `json:"geo_filter"`
		Children  []struct {
			Kind string `json:"kind"`
			Data struct {
				ApprovedAtUtc       interface{} `json:"approved_at_utc"`
				Subreddit           string      `json:"subreddit"`
				SelfText            string      `json:"selftext"`
				Url                 string      `json:"url"`
				UrlOverriddenByDest string      `json:"url_overridden_by_dest"`
				MediaMetadata       map[string]struct {
					Status string `json:"status"`
					Id     string `json:"id"`
					E      string `json:"e"`
					M      string `json:"m"`
					S      struct {
						U string `json:"u"`
						X int    `json:"x"`
						Y int    `json:"y"`
					} `json:"s"`
					P []struct {
						U string `json:"u"`
						X int    `json:"x"`
						Y int    `json:"y"`
					} `json:"p"`
				} `json:"media_metadata"`
				Preview struct {
					Enabled bool `json:"enabled"`
					Images  []struct {
						Source struct {
							Url    string `json:"url"`
							Width  int    `json:"width"`
							Height int    `json:"height"`
						} `json:"source"`
					} `json:"images"`
				} `json:"preview"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}
type CustomContext struct {
	echo.Context
	redis *redis.Client
}

func main() {

	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	e.HideBanner = true

	// Middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			redisUrl := os.Getenv("REDIS_URL")
			if redisUrl == "" {
				c.Logger().Warn("REDIS_URL not set, skipping redis connection. Use REDIS_URL to cache the image")

				return next(&CustomContext{c, nil})
			}

			redisPort := os.Getenv("REDIS_PORT")
			if redisPort == "" {
				redisPort = "6379"
			}

			redisDb := os.Getenv("REDIS_DB")
			if redisDb == "" {
				redisDb = "0"
			}

			redisDbInt, err := strconv.ParseInt(redisDb, 10, 64)
			if err != nil {
				c.Logger().Warn("REDIS_DB is not a valid integer, skipping redis connection. Use REDIS_DB to cache the image")

				return next(&CustomContext{})
			}

			cc := &CustomContext{c, redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%s", redisUrl, redisPort),
				Password: os.Getenv("REDIS_PASSWORD"),
				DB:       int(redisDbInt),
			})}

			cmd := cc.redis.Ping(context.Background())

			if cmd.Err() != nil {
				c.Logger().Warn("could not connect to redis")

				cc.redis = nil
			}

			return next(cc)
		}
	})

	e.Use(middleware.Logger())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{http.MethodGet, http.MethodOptions},
	}))

	path := os.Getenv("PATH_PREFIX")
	if path == "" {
		path = "/"
	}
	e.GET(path, func(c echo.Context) error {
		cc := c.(*CustomContext)

		var latestImageVal string
		var err error
		if cc.redis != nil {
			latestImage := cc.redis.Get(context.TODO(), "latestImage")
			latestImageVal, err = latestImage.Result()
		}

		if err != nil || latestImageVal == "" {
			c.Logger().Info("Fetching latest image")
			latestImageVal, err = getLatestImage()
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error fetching latest image")
			}

			if cc.redis != nil {
				cmd := cc.redis.Set(context.TODO(), "latestImage", latestImageVal, 600*time.Second)
				if cmd.Err() != nil {
					c.Logger().Warn("could not cache image")
				}
			}
		} else {
			c.Logger().Info("Image name fetched from cache")
		}

		var imageData string
		if cc.redis != nil {
			latestImageBin := cc.redis.Get(context.TODO(), "latestImage:bin:"+latestImageVal)
			imageData = latestImageBin.Val()
		}

		if imageData == "" {
			response, err := http.Get(latestImageVal)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error fetching image")
			}

			if response.StatusCode != http.StatusOK {
				return c.String(http.StatusInternalServerError, "Error fetching image")
			}

			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(response.Body)

			imageDataBytes, err := io.ReadAll(response.Body)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error fetching image")
			}

			imageData = string(imageDataBytes)

			imageData, err := resize.Shrink(imageData, 1200, 70)

			if err != nil {
				return c.String(http.StatusInternalServerError, "Error resizing image")
			}

			go func(data string) {
				if cc.redis == nil {
					return
				}

				_, err = cc.redis.Set(context.TODO(), "latestImage:bin:"+latestImageVal, data, 600*time.Second).Result()
				if err != nil {
					c.Logger().Warn("could not cache image")
				}
			}(imageData)
		} else {
			c.Logger().Info("Image data fetched from cache")
		}

		c.Response().Header().Set("Cache-Control", "public, max-age=600")

		return c.Blob(http.StatusOK, "image/jpeg", []byte(imageData))
	})

	e.Logger.Info("Starting server at path " + path)
	// start the server

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}

func getLatestImage() (string, error) {

	response, err := http.Get("https://www.reddit.com/r/wallpaper/.json")
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching reddit data: %s", response.Status)
	}

	jsonBytes, err := io.ReadAll(response.Body)

	redditData := redditResponse{}
	err = json.Unmarshal(jsonBytes, &redditData)
	if err != nil {
		return "", err
	}

	index := 1
	url := redditData.Data.Children[index].Data.UrlOverriddenByDest

	for strings.Contains(url, "gallery") {
		index++
		url = redditData.Data.Children[index].Data.UrlOverriddenByDest
	}

	return url, nil
}
