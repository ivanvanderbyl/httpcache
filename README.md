# `ivan.dev/httpcache`

`httpcache` is simple HTTP Client cache for Go that uses [go-redis/cache](https://github.com/go-redis/cache) as the storage layer.

It can respect cache headers to act like a private client ^^TBD^^ (e.g like a browser or API client) or
cache all responses if desired.

Responses are gzipped when stored in redis.

## Installation

```shell
go get ivan.dev/httpcache
```

## Usage

```go
// Create a new cache using Redis Cache
myCache := cache.New(&cache.Options{
Redis: redisClient,
})

// Setup transport (using tripware)
transport := tripware.New(http.DefaultTransport)
transport.Use(httpcache.WithCache(myCache, 1*time.Minute))
client := &http.Client{Transport: transport}

// Alternative setup transport using http.RoundTripper
transport := httpcache.NewCacheTransport(http.DefaultTransport, myCache, 1*time.Minute)
client := &http.Client{Transport: transport}
```

See [simple GET example](./examples/simple-get/main.go) for a runnable example.

You can inspect a response to see if it was returned from the cache using `httpcache.IsCachedResponse(resp)` or checking the value of the `X-HTTP-Cache` header.

## Configuration

### `Check`

```go
func(req *http.Request) bool
```

A function that checks if the current request is cacheable.

**Default**: All GET and HEAD requests that DO NOT specify a `range` header.

### `CacheKeyFn`

```go
func(req *http.Request) string
```

Specifies the function to generate cache keys if the current solution doesn't meet your requirements.

**Default:** `httpcache:METHOD:SHA1(url)`

## Why this exists

Two reasons:

1. You need a http client that respects cache headers and doesn't request cached assets multiple times.
2. You want to cache API responses at a low level so that your application code doesn't abuse the upstream API.

Both these use cases are not supported out of the box with `http.Client` in the Go standard library.
