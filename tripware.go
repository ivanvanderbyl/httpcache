package httpcache

import (
	"net/http"

	"github.com/go-redis/cache/v8"
	"ivan.dev/httpcache/tripware"
)

// WithCache returns a http.Client tripware that will cache responses. Use with tripware pkg.
func WithCache(c *cache.Cache, opts ...Option) tripware.Tripperware {
	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}
		return NewCacheTransport(next, RedisCache(c), opts...)
	}
}
