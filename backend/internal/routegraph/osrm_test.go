package routegraph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotravel/internal/apperr"
)

func TestOSRMRateLimitedStaleBody(t *testing.T) {
	// Reproduces the reported incident: upstream returns HTTP 429 but the
	// body is a syntactically complete (stale) OSRM route JSON. The provider
	// must surface the failure instead of accepting the stale payload.
	staleBody := `{"code":"Ok","routes":[{"distance":4242,"legs":[{"distance":4242}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(staleBody))
	}))
	defer srv.Close()

	prov := OSRMProvider{BaseURL: srv.URL}
	res, err := prov.ComputeHTTP(srv.Client(), []Point{{Lat: 30.0, Lng: 120.0}, {Lat: 30.01, Lng: 120.01}})
	if err == nil {
		t.Fatalf("expected error for 429, got result meters=%.0f provider=%s", res.TotalMeters, res.Provider)
	}
	ae, ok := err.(*apperr.Error)
	if !ok {
		t.Fatalf("expected *apperr.Error, got %T: %v", err, err)
	}
	if ae.Code != apperr.RateLimited {
		t.Fatalf("expected RATE_LIMITED code, got %s (msg=%s)", ae.Code, ae.Message)
	}
	if ae.HTTP != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429, got %d", ae.HTTP)
	}
}

func TestOSRMNonOkCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"NoRoute","routes":[]}`))
	}))
	defer srv.Close()

	prov := OSRMProvider{BaseURL: srv.URL}
	_, err := prov.ComputeHTTP(srv.Client(), []Point{{Lat: 30.0, Lng: 120.0}, {Lat: 30.01, Lng: 120.01}})
	if err == nil {
		t.Fatal("expected error for non-Ok code")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.GeoUnavailable {
		t.Fatalf("expected GEO_UNAVAILABLE, got %v", err)
	}
}

func TestOSRMHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/route/v1/driving/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":1000,"legs":[{"distance":1000}]}]}`))
	}))
	defer srv.Close()

	prov := OSRMProvider{BaseURL: srv.URL}
	res, err := prov.ComputeHTTP(srv.Client(), []Point{{Lat: 30.0, Lng: 120.0}, {Lat: 30.01, Lng: 120.01}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalMeters != 1000 {
		t.Fatalf("expected 1000m, got %.0f", res.TotalMeters)
	}
	if res.Provider != "osrm" {
		t.Fatalf("expected provider osrm, got %s", res.Provider)
	}
}

func TestOSRM5xxBodyIgnored(t *testing.T) {
	// Even when the upstream 5xx body is a complete route JSON, the status
	// code must win and the body must not be trusted.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":9999,"legs":[{"distance":9999}]}]}`))
	}))
	defer srv.Close()

	prov := OSRMProvider{BaseURL: srv.URL}
	_, err := prov.ComputeHTTP(srv.Client(), []Point{{Lat: 30.0, Lng: 120.0}, {Lat: 30.01, Lng: 120.01}})
	if err == nil {
		t.Fatal("expected error for 502")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.GeoUnavailable {
		t.Fatalf("expected GEO_UNAVAILABLE, got %v", err)
	}
}
