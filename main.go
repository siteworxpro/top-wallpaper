package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Kind  string `json:"kind"`
				Title string `json:"title"`
				URL   string `json:"url"`
				Data  struct {
				} `json:"data"`
			}
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

				return next(&CustomContext{})
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

	path := os.Getenv("PATH")
	if path == "" {
		path = "/"
	}
	e.GET(path, func(c echo.Context) error {
		cc := c.(*CustomContext)
		latestImage := cc.redis.Get(context.TODO(), "latestImage")
		latestImageVal, err := latestImage.Result()

		if err != nil || latestImageVal == "" {
			c.Logger().Info("Fetching latest image")
			latestImageVal = getLatestImage()
			cmd := cc.redis.Set(context.TODO(), "latestImage", latestImageVal, 600*time.Second)
			if cmd.Err() != nil {
				c.Logger().Warn("could not cache image")
			}
		} else {
			c.Logger().Info("Image name fetched from cache")
		}

		latestImageBin := cc.redis.Get(context.TODO(), "latestImage:bin:"+latestImageVal)
		imageData := latestImageBin.Val()
		if imageData == "" {
			response, err := http.Get(latestImageVal)
			if err != nil {
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

			go func(data string) {
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

func getLatestImage() string {
	response, err := http.Get("https://www.reddit.com/r/wallpaper/.json")
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	jsonBytes, err := io.ReadAll(response.Body)

	redditData := redditResponse{}
	err = json.Unmarshal(jsonBytes, &redditData)
	if err != nil {
		panic(err)
	}

	url := redditData.Data.Children[1].Data.URL

	return url
}
