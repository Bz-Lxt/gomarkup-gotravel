package geo

import "math"

const EarthRadiusM = 6371000.0

func ValidCoord(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func ClampCoord(lat, lng float64) (float64, float64) {
	lat = math.Round(lat*1e7) / 1e7
	lng = math.Round(lng*1e7) / 1e7
	return lat, lng
}

func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusM * c
}

func PolylineMeters(pts [][2]float64) float64 {
	var sum float64
	for i := 1; i < len(pts); i++ {
		sum += Haversine(pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1])
	}
	return sum
}

func Centroid(pts [][2]float64) (lat, lng float64) {
	if len(pts) == 0 {
		return 0, 0
	}
	for _, p := range pts {
		lat += p[0]
		lng += p[1]
	}
	n := float64(len(pts))
	return lat / n, lng / n
}

func Interpolate(lat1, lng1, lat2, lng2, t float64) (float64, float64) {
	if t <= 0 {
		return lat1, lng1
	}
	if t >= 1 {
		return lat2, lng2
	}
	return lat1 + (lat2-lat1)*t, lng1 + (lng2-lng1)*t
}
