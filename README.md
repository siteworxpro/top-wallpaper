# Top Wallpaper Image

A simple app that fetches the top wallpaper image from [/r/wallpaper](https://reddit.com/r/wallpaper) and serves it as a wallpaper.

## Installation

```bash
go mod tidy
go build
```

## Usage

```bash
./top-wallpaper
```

[http://localhost:8080](http://localhost:8080) will serve the top wallpaper image.

Available environment variables:
```
REDIS_URL: Redis URL (default: localhost)
REDIS_PORT: Redis port (default: 6379)
REDIS_PASSWORD: Redis password (default: "")
REDIS_DB: Redis database (default: 0)
ALLOWED_ORIGINS: Allowed origins for CORS (default: "")
PATH: Path to serve the image (default: /)
PORT: Port to serve the image (default: 8080)
```

## docker
```shell
docker run -d -p 8080:8080 --name top-wallpaper -e REDIS_URL=redis -e REDIS_PORT=6379 -e REDIS_PASSWORD=pass -e REDIS_DB=0 -e ALLOWED_ORIGINS=http://localhost:8080 -e PATH=/wallpaper -e PORT=8080 --network=host --restart=always siteworxpro/top-wallpaper:latest
```

## License

[MIT](https://choosealicense.com/licenses/mit/)
```
MIT License

Copyright (c) 2025 Siteworx Professionals, LLC

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
