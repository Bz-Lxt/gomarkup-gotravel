package routegraph

import "gotravel/internal/geo"

type Point struct {
	Lat float64
	Lng float64
}

type Segment struct {
	FromIdx int     `json:"from_idx"`
	ToIdx   int     `json:"to_idx"`
	Meters  float64 `json:"meters"`
}

type DistanceResult struct {
	TotalMeters float64   `json:"total_meters"`
	Segments    []Segment `json:"segments"`
	ETAMinutes  float64   `json:"eta_minutes"`
	WalkKmh     float64   `json:"walk_kmh"`
	Provider    string    `json:"provider"`
}

type Provider interface {
	Compute(points []Point) (DistanceResult, error)
	Name() string
}

type HaversineProvider struct{}

func (HaversineProvider) Name() string { return "haversine" }

func (HaversineProvider) Compute(points []Point) (DistanceResult, error) {
	if len(points) == 0 {
		return DistanceResult{Segments: []Segment{}, WalkKmh: 4.5, Provider: "haversine"}, nil
	}
	// Allocate a fresh slice per call. A package-level shared buffer would alias
	// across sequential/concurrent calls: a later Compute would overwrite the
	// segment slots still referenced by an already-returned DistanceResult,
	// making route A's segments mutate into route B's ("串单"). Each result must
	// own an independent backing array so returned values stay stable.
	segments := make([]Segment, 0, len(points)-1)
	res := DistanceResult{Segments: segments, WalkKmh: 4.5, Provider: "haversine"}
	for i := 1; i < len(points); i++ {
		d := geo.Haversine(points[i-1].Lat, points[i-1].Lng, points[i].Lat, points[i].Lng)
		res.Segments = append(res.Segments, Segment{FromIdx: i - 1, ToIdx: i, Meters: d})
		res.TotalMeters += d
	}
	if res.WalkKmh > 0 {
		res.ETAMinutes = (res.TotalMeters / 1000) / res.WalkKmh * 60
	}
	return res, nil
}

type OSRMProvider struct {
	BaseURL string
}

func (o OSRMProvider) Name() string { return "osrm" }

type unavail struct{ s string }

func (e unavail) Error() string { return e.s }

func fmtUnavailable(s string) error { return unavail{s} }
