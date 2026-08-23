package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gotravel/internal/config"
	"gotravel/internal/handler"
	"gotravel/internal/ws"
)

type gatedResponseWriter struct {
	header  http.Header
	status  int
	body    bytes.Buffer
	entered chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func newGatedResponseWriter(release <-chan struct{}) *gatedResponseWriter {
	return &gatedResponseWriter{
		header:  make(http.Header),
		entered: make(chan struct{}),
		release: release,
	}
}

func (w *gatedResponseWriter) Header() http.Header {
	return w.header
}

func (w *gatedResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *gatedResponseWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return w.body.Write(p)
}

func TestConcurrentMetricsAndHealthResponsesRemainIsolated(t *testing.T) {
	hub := ws.NewHub()
	t.Cleanup(hub.Stop)
	hub.Metrics.Inbound.Store(123456789)
	hub.Metrics.Outbound.Store(234567890)
	hub.Metrics.Naive.Store(345678901)

	api := &handler.API{
		Cfg: config.Config{
			GPSProvider:   "sim",
			RouteProvider: "haversine",
		},
		Hub:     hub,
		GeoName: "rtree",
	}
	routes := api.Routes()

	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)

	metricsWriter := newGatedResponseWriter(release)
	metricsDone := make(chan struct{})
	go func() {
		routes.ServeHTTP(metricsWriter, httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil))
		close(metricsDone)
	}()

	select {
	case <-metricsWriter.entered:
	case <-time.After(2 * time.Second):
		unblock()
		<-metricsDone
		t.Fatal("metrics response never reached the writer")
	}

	healthWriter := httptest.NewRecorder()
	routes.ServeHTTP(healthWriter, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	unblock()

	select {
	case <-metricsDone:
	case <-time.After(2 * time.Second):
		t.Fatal("metrics response did not finish after the writer was released")
	}

	if healthWriter.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthWriter.Code, http.StatusOK)
	}
	if metricsWriter.status != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", metricsWriter.status, http.StatusOK)
	}

	var got struct {
		OK   bool `json:"ok"`
		Data struct {
			Inbound  int64 `json:"inbound"`
			Outbound int64 `json:"outbound"`
		} `json:"data"`
	}
	if err := json.Unmarshal(metricsWriter.body.Bytes(), &got); err != nil {
		t.Fatalf("metrics response is not valid JSON: %v; body=%q", err, metricsWriter.body.Bytes())
	}
	if !got.OK || got.Data.Inbound != 123456789 || got.Data.Outbound != 234567890 {
		t.Fatalf("metrics response was contaminated: %+v", got)
	}
}
