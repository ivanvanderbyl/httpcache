package tripware

import (
	"net/http"
)

type HeaderRoundTripper struct {
	next   http.RoundTripper
	Header http.Header
}

// WithHeader adds the given headers to each request.
// Example: roundtrip.Use(WithHeader(http.Header{"Authorization": []string{"Bearer 1234567890"}}))
func WithHeader(Header http.Header) Tripperware {
	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}
		return &HeaderRoundTripper{
			next:   next,
			Header: Header,
		}
	}
}

func (rt *HeaderRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if rt.Header != nil {
		for k, v := range rt.Header {
			req.Header[k] = v
		}
	}
	return rt.next.RoundTrip(req)
}
