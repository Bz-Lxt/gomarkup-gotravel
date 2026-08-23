package routegraph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gotravel/internal/apperr"
)

// Compute calls a real OSRM route service when BaseURL is set.
// Example: OSRM_BASE_URL=http://osrm:5000
func (o OSRMProvider) ComputeHTTP(client *http.Client, points []Point) (DistanceResult, error) {
	if strings.TrimSpace(o.BaseURL) == "" {
		return DistanceResult{}, fmtUnavailable("osrm base url empty")
	}
	if len(points) < 2 {
		return DistanceResult{Segments: []Segment{}, Provider: "osrm", WalkKmh: 4.5}, nil
	}
	coords := make([]string, len(points))
	for i, p := range points {
		coords[i] = fmt.Sprintf("%.6f,%.6f", p.Lng, p.Lat)
	}
	url := strings.TrimRight(o.BaseURL, "/") + "/route/v1/driving/" + strings.Join(coords, ";") + "?overview=false&annotations=distance"
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Get(url)
	if err != nil {
		return DistanceResult{}, apperr.New(http.StatusBadGateway, apperr.GeoUnavailable, "osrm: "+err.Error())
	}
	defer resp.Body.Close()
	// The upstream status code is authoritative. A rate-limited (429) or
	// otherwise failing response must not be accepted even when the body
	// happens to be syntactically valid JSON: OSRM occasionally returns 429
	// carrying a stale, otherwise well-formed route payload. Decoding the
	// body first and then gating on parsed.Code/parsed.Routes let that stale
	// payload masquerade as a fresh result, so the trip distance got updated
	// with outdated numbers and the client saw a 200. Reject by status before
	// trusting anything in the body.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DistanceResult{}, osrmHTTPError(resp.StatusCode)
	}
	var parsed struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Legs     []struct {
				Distance float64 `json:"distance"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return DistanceResult{}, apperr.New(http.StatusBadGateway, apperr.GeoUnavailable, "osrm decode: "+err.Error())
	}
	if parsed.Code != "Ok" || len(parsed.Routes) == 0 {
		return DistanceResult{}, apperr.New(http.StatusBadGateway, apperr.GeoUnavailable, "osrm code "+parsed.Code)
	}
	res := DistanceResult{Segments: []Segment{}, WalkKmh: 4.5, Provider: "osrm", TotalMeters: parsed.Routes[0].Distance}
	for i, leg := range parsed.Routes[0].Legs {
		res.Segments = append(res.Segments, Segment{FromIdx: i, ToIdx: i + 1, Meters: leg.Distance})
	}
	if res.WalkKmh > 0 {
		res.ETAMinutes = (res.TotalMeters / 1000) / res.WalkKmh * 60
	}
	return res, nil
}

// osrmHTTPError maps a non-2xx upstream status to a client-facing error. A
// rate-limit (429) is surfaced as RATE_LIMITED so callers can distinguish it
// from a generic upstream outage; any other failure becomes GEO_UNAVAILABLE.
// The response body is intentionally not parsed -- it cannot be trusted.
func osrmHTTPError(status int) error {
	if status == http.StatusTooManyRequests {
		return apperr.New(http.StatusTooManyRequests, apperr.RateLimited, "osrm rate limited")
	}
	return apperr.New(http.StatusBadGateway, apperr.GeoUnavailable, fmt.Sprintf("osrm status %d", status))
}

func (o OSRMProvider) Compute(points []Point) (DistanceResult, error) {
	return o.ComputeHTTP(nil, points)
}
