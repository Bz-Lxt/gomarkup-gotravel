package routegraph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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
		return DistanceResult{}, fmt.Errorf("osrm: %w", err)
	}
	defer resp.Body.Close()
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
		return DistanceResult{}, fmt.Errorf("osrm decode: %w", err)
	}
	if parsed.Code != "Ok" || len(parsed.Routes) == 0 {
		if resp.StatusCode >= 400 {
			return DistanceResult{}, fmt.Errorf("osrm status %d", resp.StatusCode)
		}
		return DistanceResult{}, fmt.Errorf("osrm code %s", parsed.Code)
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

func (o OSRMProvider) Compute(points []Point) (DistanceResult, error) {
	return o.ComputeHTTP(nil, points)
}
