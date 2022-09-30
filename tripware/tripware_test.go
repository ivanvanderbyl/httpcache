package tripware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/motemen/go-loghttp"
	"github.com/stretchr/testify/require"
)

type exampleTripper struct {
	base http.RoundTripper
}

var _ http.RoundTripper = &exampleTripper{}

func (t *exampleTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req)
}

func TestTripware(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") == "Bearer 1234567890" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))

	defer testServer.Close()

	roundtrip := New(http.DefaultTransport)

	// Middleware 1
	roundtrip.Use(func(rt http.RoundTripper) http.RoundTripper {
		return &loghttp.Transport{Transport: rt}
	})

	// Middleware 2
	roundtrip.Use(func(rt http.RoundTripper) http.RoundTripper {
		return &exampleTripper{base: rt}
	})

	// Middleware 3
	roundtrip.Use(WithHeader(http.Header{"Authorization": []string{"Bearer 1234567890"}}))

	client := &http.Client{Transport: roundtrip}

	require.Equal(t, 3, len(roundtrip.items))

	req, err := http.NewRequest("GET", testServer.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}
