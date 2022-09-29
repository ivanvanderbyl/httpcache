package tripware

import (
	"net/http"
)

type (
	Tripware struct {
		base  http.RoundTripper
		items []Tripperware
	}

	// Middleware func(rt http.RoundTripper) http.RoundTripper
	Tripperware func(rt http.RoundTripper) http.RoundTripper

	// RoundTrip wraps a func to make it into a http.RoundTripper. Similar to http.HandleFunc.
	RoundTrip func(req *http.Request) (*http.Response, error)
)

func NewRoundTripper(original http.RoundTripper) http.RoundTripper {
	if original == nil {
		original = http.DefaultTransport
	}

	return RoundTrip(func(request *http.Request) (*http.Response, error) {
		response, err := original.RoundTrip(request)
		return response, err
	})
}

func (f RoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func New(base http.RoundTripper) *Tripware {
	return &Tripware{base: base, items: []Tripperware{}}
}

func (t *Tripware) Use(middleware Tripperware) {
	t.items = append(t.items, middleware)
}

func (t *Tripware) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.base == nil {
		t.base = http.DefaultTransport
	}

	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}

	for i := len(t.items) - 1; i >= 0; i-- {
		transport = t.items[i](transport)
	}

	return transport.RoundTrip(req)
}
