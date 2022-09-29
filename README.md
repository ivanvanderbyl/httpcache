# httpcache

A simple HTTP Client cache for Go that uses [go-redis/cache](https://github.com/go-redis/cache) as the storage layer. It can respect cache headers or acts as a private client (e.g like a browser or API client)

## Why this exists

The standard HTTP client in Go doesn't have any concept of response caching, yet it's quite common to pass through requests from your app to a 3rd-party API after adding authentication or other request parameters, but you don't want to abuse that API or exceed your rate limit, so you implement a cache so that the same request returns the same response for a short period of time e.g. 30 minutes.
