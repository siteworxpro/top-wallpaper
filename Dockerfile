FROM siteworxpro/golang:1.24.0 AS build

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

RUN  adduser -u 1001 -g appuser appuser -D && \
    chown -R appuser:appuser /app

USER 1001

ENTRYPOINT ["/app/top-wallpaper"]