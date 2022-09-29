package httpcache

import (
	"time"
)

type (
	Cache interface {
		Set(key string, data []byte)
		Get(key string) ([]byte, bool)
		Del(key string)
	}

	RedisCache struct {
		TTL time.Duration
	}
)

func (c *RedisCache) Set(key string, data []byte) {

}
