FROM siteworxpro/golang:1.23.5 AS build

WORKDIR /app

ADD . .

ENV GOPRIVATE=git.s.int
ENV GOPROXY=direct
ENV CGO_ENABLED=0

RUN go mod tidy && go build -o top-wallpaper .

FROM alpine:latest AS runtime

EXPOSE 8080

WORKDIR /app

COPY --from=build /app/top-wallpaper /app/top-wallpaper

ENTRYPOINT ["/app/top-wallpaper"]