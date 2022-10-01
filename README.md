# `ivan.dev/httpcache`

`httpcache` is simple `net/http` client cache for Go that caches responses in your chosen cache backend.

**Features:**

- [x] Configurable cache keys
- [x] Configurable TTL
- [x] Configurable caching check to decide if a response should be cached
- [x] Configurable cache backends
- [x] Configurable compression
- [x] Memory only cache backend
- [x] Redis Cache backend, with support for multi-tiered caching
- [ ] RFC 7234 support (PRs welcome)

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
// You can use other cache implementations with this approach
transport := httpcache.NewCacheTransport(http.DefaultTransport, httpcache.RedisCache(myCache), 1*time.Minute)
client := &http.Client{Transport: transport}
```

See [simple GET example](./examples/simple-get/main.go) for a runnable example.

You can inspect a response to see if it was returned from the cache using `httpcache.IsCachedResponse(resp)` or checking the value of the `X-HTTP-Cache` header.

## Configuration

### `WithTTL(time.Duration)`

Sets the default TTL (Time-To-Live) for each cached request, if the underlying cache implementation supports per-item TTLs. If you're using the default `RedisCache` implementation, then this value will be used, however, if you're using the `TinyLFU` implementation, the TTL will be configured by `TinyLFU`

### `WithRequestChecker(...)`

```go
WithRequestChecker(func(req *http.Request) bool{
  return req.Method == http.MethodGet
})
```

A function that checks if the current request is cacheable.

**Default**: All GET and HEAD requests that DO NOT specify a `range` header.

### `WithCacheKeyFn(...)`

Specifies the function to generate cache keys if the current solution doesn't meet your requirements.

Default implementation looks like this and produces keys like `"httpcache:GET:89dce6a446a69d6b9bdc01ac75251e4c322bcdff"`

```go
WithCacheKeyFn(func(req *http.Request) string{
  urlString := req.URL.String()
  return fmt.Sprintf("%s:%s:%x", cacheKeyPredix, req.Method, sha1.Sum([]byte(urlString)))
})
```

### `WithCompression()`

Enables extra GZIP compression of each response. This usually ins't necessary since with Redis Cache because it has its own S2 based caching if it decides it is needed.

Use this if the cache implementation doesn't support compression and you need it.

## Why this exists

Two reasons:

1. You need a http client that respects cache headers and doesn't request cached assets multiple times.
2. You want to cache API responses at a low level so that your application code doesn't abuse the upstream API.

Both these use cases are not supported out of the box with `http.Client` in the Go standard library.
