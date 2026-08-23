package routegraph_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"gotravel/internal/routegraph"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOSRMProviderRejectsHTTPErrorWithRouteBody(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"code":"Ok","routes":[{"distance":1250,"legs":[{"distance":1250}]}]}`,
			)),
			Request: r,
		}, nil
	})}

	provider := routegraph.OSRMProvider{BaseURL: "https://route.example"}
	result, err := provider.ComputeHTTP(client, []routegraph.Point{
		{Lat: 31.2304, Lng: 121.4737},
		{Lat: 31.2200, Lng: 121.4800},
	})
	if err == nil {
		t.Fatalf("expected HTTP 429 to be returned as an error, got successful distance %.0fm", result.TotalMeters)
	}
}
