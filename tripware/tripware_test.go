package tripware

import (
	"net/http"
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
	roundtrip := New(http.DefaultTransport)

	// Middleware 1
	roundtrip.Use(func(rt http.RoundTripper) http.RoundTripper {
		return &loghttp.Transport{Transport: rt}
	})

	// Middleware 2
	roundtrip.Use(func(rt http.RoundTripper) http.RoundTripper {
		return &exampleTripper{base: rt}
	})

	client := &http.Client{Transport: roundtrip}

	require.Equal(t, 2, len(roundtrip.items))

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}
