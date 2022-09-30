package httpcache

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-redis/cache/v8"
)

const cacheKeyPredix = "httpcache:"
const HTTPCacheHeader = "X-Http-Cache"

type (
	// Transport is an implementation of http.RoundTripper that will return values from a cache
	// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
	// to repeated requests allowing servers to return 304 / Not Modified
	Transport struct {
		// The RoundTripper interface actually used to make requests
		// If nil, http.DefaultTransport is used
		Cache *cache.Cache
		next  http.RoundTripper
	}
)

var _ http.RoundTripper = &Transport{}

func NewCacheTransport(next http.RoundTripper, c *cache.Cache) *Transport {
	return &Transport{next: next, Cache: c}
}

// RoundTrip implements the RoundTripper interface
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	key := cacheKey(req)

	if t.Cache.Exists(ctx, key) {
		var respBytes []byte
		if err := t.Cache.Get(ctx, key, &respBytes); err != nil {
			return nil, err
		}

		resp, err := hydrateResponse(req, respBytes)
		if err != nil {
			return nil, err
		}

		resp.Header.Add(HTTPCacheHeader, "HIT")
		return resp, nil
	}

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	dumpedResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return resp, err
	}

	log.Println(string(dumpedResponse))

	item := &cache.Item{
		Ctx:   ctx,
		Key:   key,
		TTL:   1 * time.Minute,
		Value: dumpedResponse,
	}

	if err := t.Cache.Set(item); err != nil {
		return nil, err
	}

	resp.Header.Add(HTTPCacheHeader, "MISS")
	return resp, nil
}

func IsCachedResponse(resp *http.Response) bool {
	return resp.Header.Get(HTTPCacheHeader) == "HIT"
}

func cacheKey(req *http.Request) string {
	return url.PathEscape(cacheKeyPredix + req.Method + ":" + req.URL.String())
}

func hydrateResponse(req *http.Request, b []byte) (*http.Response, error) {
	buf := bytes.NewBuffer(b)
	return http.ReadResponse(bufio.NewReader(buf), req)
}
