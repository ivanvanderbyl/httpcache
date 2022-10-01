package httpcache

import (
	"context"
	"time"

	"github.com/go-redis/cache/v8"
)

type (
	Cache interface {
		Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
		Get(ctx context.Context, key string) ([]byte, error)
		Del(ctx context.Context, key string) error
	}

	redisCache struct {
		cache *cache.Cache
	}

	tinyLFUCache struct {
		cache *cache.TinyLFU
	}
)

var _ Cache = (*redisCache)(nil)
var _ Cache = (*tinyLFUCache)(nil)

// RedisCache is a cache implementation using Redis Cache. It can be constructed with
// an LFU cache to do memory hits before hitting Redis.
func RedisCache(cache *cache.Cache) Cache {
	return &redisCache{cache: cache}
}

// MemoryCache is a simple in-memory cache using an LFU cache.
func MemoryCache(size int, ttl time.Duration) Cache {
	cache := cache.NewTinyLFU(1000, 0)
	return &tinyLFUCache{cache: cache}
}

func (r *redisCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return r.cache.Set(&cache.Item{
		Key:   key,
		Value: data,
		TTL:   ttl,
		Ctx:   ctx,
	})
}

func (r *redisCache) Get(ctx context.Context, key string) (respBytes []byte, err error) {
	respBytes = make([]byte, 0)
	if err := r.cache.Get(ctx, key, &respBytes); err != nil {
		return nil, err
	}
	return respBytes, nil
}

func (r *redisCache) Del(ctx context.Context, key string) error {
	return r.cache.Delete(ctx, key)
}

func (t *tinyLFUCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	t.cache.Set(key, data)
	return nil
}

func (t *tinyLFUCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, ok := t.cache.Get(key)
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	return data, nil
}

func (t *tinyLFUCache) Del(ctx context.Context, key string) error {
	t.cache.Del(key)
	return nil
}
