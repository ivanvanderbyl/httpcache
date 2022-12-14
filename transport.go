package httpcache

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httputil"
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
		Cache             Cache
		check             func(req *http.Request) bool
		cacheKeyFn        func(req *http.Request) string
		defaultTTL        time.Duration
		enableCompression bool
		next              http.RoundTripper
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

func WithCompression() Option {
	return func(t *Transport) {
		t.enableCompression = true
	}
}

var _ http.RoundTripper = &Transport{}

func NewCacheTransport(next http.RoundTripper, c Cache, options ...Option) *Transport {
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

	respBytes, err := t.Cache.Get(ctx, key)
	if err != nil {
		if err != cache.ErrCacheMiss {
			return nil, err
		}
	}

	if len(respBytes) > 0 {
		// Cache Hit
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

	// Cache Miss, make fresh request
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	dumpedResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return resp, err
	}

	if t.enableCompression {
		dumpedResponse, err = compressResponseDump(dumpedResponse)
		if err != nil {
			return nil, err
		}
	}

	if err := t.Cache.Set(ctx, key, dumpedResponse, t.defaultTTL); err != nil {
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
	return fmt.Sprintf("%s:%s:%x", cacheKeyPredix, req.Method, sha1.Sum([]byte(urlString)))
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
	if len(dumpedResponse) == 0 {
		return dumpedResponse, nil
	}

	// Detect if the response is compressed using gzip
	if dumpedResponse[0] != 31 || dumpedResponse[1] != 139 {
		return dumpedResponse, nil
	}

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
