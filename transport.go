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

const hit = "HIT"
const miss = "MISS"

type (
	// Transport is an implementation of http.RoundTripper that will return values from a cache
	// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
	// to repeated requests allowing servers to return 304 / Not Modified
	Transport struct {
		// The RoundTripper interface actually used to make requests
		// If nil, http.DefaultTransport is used
		Cache *cache.Cache
		Check func(req *http.Request) bool
		TTL   time.Duration
		next  http.RoundTripper
	}
)

var _ http.RoundTripper = &Transport{}

func NewCacheTransport(next http.RoundTripper, c *cache.Cache, ttl time.Duration) *Transport {
	return &Transport{next: next, Cache: c, TTL: ttl, Check: DefaultRequestChecker}
}

// RoundTrip implements the RoundTripper interface
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.Check(req) || t.TTL == 0 {
		return t.next.RoundTrip(req)
	}

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

		resp.Header.Add(HTTPCacheHeader, hit)
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
		TTL:   t.TTL,
		Value: dumpedResponse,
	}

	if err := t.Cache.Set(item); err != nil {
		return nil, err
	}

	resp.Header.Add(HTTPCacheHeader, miss)
	return resp, nil
}

func IsCachedResponse(resp *http.Response) bool {
	return resp.Header.Get(HTTPCacheHeader) == hit
}

func DefaultRequestChecker(req *http.Request) bool {
	return (req.Method == http.MethodGet || req.Method == http.MethodHead) && req.Header.Get("range") == ""
}

func cacheKey(req *http.Request) string {
	return url.PathEscape(cacheKeyPredix + req.Method + ":" + req.URL.String())
}

func hydrateResponse(req *http.Request, b []byte) (*http.Response, error) {
	buf := bytes.NewBuffer(b)
	return http.ReadResponse(bufio.NewReader(buf), req)
}
