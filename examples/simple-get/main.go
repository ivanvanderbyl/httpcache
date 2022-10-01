package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"ivan.dev/httpcache"
	"ivan.dev/httpcache/tripware"
)

func main() {
	// Connect to redis on localhost
	redisClient := redis.NewClient(&redis.Options{})

	// Create a new cache
	myCache := cache.New(&cache.Options{
		Redis: redisClient,
	})

	// Setup transport
	transport := tripware.New(http.DefaultTransport)
	transport.Use(
		httpcache.WithCache(
			myCache,
			httpcache.WithTTL(1*time.Minute),
			httpcache.WithCompression(),
		),
	)

	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	t1 := time.Now()
	// Make request the usual way
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	if httpcache.IsCachedResponse(resp) {
		log.Println("Retrieved from cache in", time.Since(t1))
	} else {
		log.Println("Retrieved from server in", time.Since(t1))
	}

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	log.Println(string(body[0:100]))
}
