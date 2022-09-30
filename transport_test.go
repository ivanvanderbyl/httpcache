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

	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr: "localhost:6379",
	// })

	redisClient, redisMock := redismock.NewClientMock()

	req, err := http.NewRequest("GET", testServer.URL, nil)
	require.NoError(t, err)
	redisMock.Regexp().ExpectSet(cacheKey(req), `.+`, 1*time.Minute).SetVal("OK")
	redisMock.ExpectGet(cacheKey(req)).SetVal("hello world")

	mycache := cache.New(&cache.Options{
		Redis:        redisClient,
		LocalCache:   cache.NewTinyLFU(1000, time.Minute),
		StatsEnabled: true,
	})

	cacheTransport := NewCacheTransport(http.DefaultTransport, mycache)
	client := http.Client{Transport: cacheTransport}

	resp1, err := client.Do(req.Clone(ctx))
	require.NoError(t, err)
	require.Equal(t, 200, resp1.StatusCode)

	resp2, err := client.Do(req.Clone(ctx))
	require.NoError(t, err)
	require.Equal(t, 200, resp2.StatusCode)
	require.Equal(t, true, IsCachedResponse(resp2))

}
