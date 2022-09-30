# tripware

Tripware provides a simple way to compose multiple `http.RoundTrip`s against a single `http.Client`, similar to middleware in a server context.

## Installation

```shell
go get ivan.dev/httpcache/tripware
```

## Usage

`tripware.Tripware` will apply all roundtrippers in the order specified when the client makes a request.

```go
roundtrip:= tripware.New(http.DefaultTransport)

// Middleware 1
roundtrip.Use(func(rt http.RoundTripper) http.RoundTripper {
return &loghttp.Transport{Transport: rt}
})

// Middleware 2 adds the authorization header to each request.
roundtrip.Use(tripware.WithHeader(http.Header{"Authorization": []string{"Bearer 1234567890"}}))

// Apply the roundtrip transport to your custom http.Client
client := &http.Client{Transport: roundtrip}

req:= http.NewRequest(...)
client.Do(req)
```

See [`headers.go`](./headers.go) for an example of how to write a Triperware that adds headers to each request.
