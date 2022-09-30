package httpcache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/require"
)

func TestCacheTransport(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
		w.Header().Set("Cache-Control", "max-age=10")
	}))

	defer testServer.Close()
	ctx := context.TODO()

	redisClient, _ := redismock.NewClientMock()
	mycache := cache.New(&cache.Options{
		Redis:      redisClient,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})

	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr: "localhost:6379",
	// })

	t.Run("It caches GET requests", func(t *testing.T) {
		redisClient, redisMock := redismock.NewClientMock()
		req, err := http.NewRequestWithContext(ctx, "GET", testServer.URL, nil)
		require.NoError(t, err)
		redisMock.Regexp().ExpectSet(cacheKey(req), `.+`, 1*time.Minute).SetVal("OK")
		redisMock.ExpectGet(cacheKey(req)).SetVal("hello world")

		mycache := cache.New(&cache.Options{
			Redis:        redisClient,
			LocalCache:   cache.NewTinyLFU(1000, time.Minute),
			StatsEnabled: true,
		})

		cacheTransport := NewCacheTransport(http.DefaultTransport, mycache, 1*time.Minute)
		client := &http.Client{Transport: cacheTransport}
		first, second := expectCachedResponse(client, req)
		require.Equal(t, false, first)
		require.Equal(t, true, second)
	})

	t.Run("It does not cache POST", func(t *testing.T) {
		redisClient, _ := redismock.NewClientMock()
		req, err := http.NewRequest("POST", testServer.URL, nil)
		require.NoError(t, err)

		mycache := cache.New(&cache.Options{
			Redis:      redisClient,
			LocalCache: cache.NewTinyLFU(1000, time.Minute),
		})

		cacheTransport := NewCacheTransport(http.DefaultTransport, mycache, 10*time.Second)
		client := &http.Client{Transport: cacheTransport}
		first, second := expectCachedResponse(client, req)
		require.Equal(t, false, first)
		require.Equal(t, false, second)
	})

	t.Run("It does not cache without TTL", func(t *testing.T) {
		redisClient, _ := redismock.NewClientMock()
		req, err := http.NewRequest("GET", testServer.URL, nil)
		require.NoError(t, err)

		mycache := cache.New(&cache.Options{
			Redis:      redisClient,
			LocalCache: cache.NewTinyLFU(1000, time.Minute),
		})

		cacheTransport := NewCacheTransport(http.DefaultTransport, mycache, 0)
		client := &http.Client{Transport: cacheTransport}
		first, second := expectCachedResponse(client, req)
		require.Equal(t, false, first)
		require.Equal(t, false, second)
	})

	t.Run("It does not cache ranged requests", func(t *testing.T) {
		req, err := http.NewRequest("GET", testServer.URL, nil)
		req.Header.Add("range", "bytes=0-10")
		require.NoError(t, err)

		cacheTransport := NewCacheTransport(http.DefaultTransport, mycache, 10*time.Second)
		client := &http.Client{Transport: cacheTransport}
		first, second := expectCachedResponse(client, req)
		require.Equal(t, false, first)
		require.Equal(t, false, second)
	})
}

func expectCachedResponse(client *http.Client, req *http.Request) (bool, bool) {
	resp1, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	resp2, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return IsCachedResponse(resp1), IsCachedResponse(resp2)
}
