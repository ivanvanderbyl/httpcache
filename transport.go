package httpcache

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
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
		Cache      *cache.Cache
		check      func(req *http.Request) bool
		cacheKeyFn func(req *http.Request) string
		defaultTTL time.Duration
		next       http.RoundTripper
	}

	Option func(*Transport)
)

// WithCache returns a new RoundTripper that will cache responses
func WithTTL(ttl time.Duration) Option {
	return func(t *Transport) {
		t.defaultTTL = ttl
	}
}

// WithRequestChecker returns a new RoundTripper that will check if a request should be cached
func WithRequestChecker(check func(req *http.Request) bool) Option {
	return func(t *Transport) {
		t.check = check
	}
}

// WithCacheKeyFn returns a new RoundTripper that will use a custom function to generate cache keys
func WithCacheKeyFn(fn func(req *http.Request) string) Option {
	return func(t *Transport) {
		t.cacheKeyFn = fn
	}
}

var _ http.RoundTripper = &Transport{}

func NewCacheTransport(next http.RoundTripper, c *cache.Cache, options ...Option) *Transport {
	if next == nil {
		next = http.DefaultTransport
	}

	t := &Transport{
		next:       next,
		Cache:      c,
		defaultTTL: time.Second * 30,
		check:      DefaultRequestChecker,
		cacheKeyFn: CacheKey,
	}

	for _, option := range options {
		option(t)
	}

	return t
}

// RoundTrip implements the RoundTripper interface
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.check(req) || t.defaultTTL == 0 {
		return t.next.RoundTrip(req)
	}

	ctx := req.Context()
	key := t.cacheKeyFn(req)

	if t.Cache.Exists(ctx, key) {
		var respBytes []byte
		var err error
		if err := t.Cache.Get(ctx, key, &respBytes); err != nil {
			return nil, err
		}

		respBytes, err = decompressResponseDump(respBytes)
		if err != nil {
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

	dumpedResponse, err = compressResponseDump(dumpedResponse)
	if err != nil {
		return nil, err
	}

	log.Printf("Compressed payload is %d bytes", len(dumpedResponse))

	item := &cache.Item{
		Ctx:   ctx,
		Key:   key,
		TTL:   t.defaultTTL,
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

func CacheKey(req *http.Request) string {
	urlString := req.URL.String()
	hashedURL := fmt.Sprintf("%x", sha1.Sum([]byte(urlString)))
	return url.PathEscape(cacheKeyPredix + req.Method + ":" + hashedURL)
}

func hydrateResponse(req *http.Request, b []byte) (*http.Response, error) {
	buf := bytes.NewBuffer(b)
	return http.ReadResponse(bufio.NewReader(buf), req)
}

func compressResponseDump(dumpedResponse []byte) ([]byte, error) {
	var buf bytes.Buffer

	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(dumpedResponse); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressResponseDump(dumpedResponse []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(dumpedResponse))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
