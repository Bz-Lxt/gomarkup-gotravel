package geo

import "testing"

func TestBeijingShanghai(t *testing.T) {
	// Beijing ~39.9042,116.4074  Shanghai ~31.2304,121.4737  ~1068km
	d := Haversine(39.9042, 116.4074, 31.2304, 121.4737)
	ref := 1067600.0
	rel := (d - ref) / ref
	if rel < 0 {
		rel = -rel
	}
	if rel > 0.005 {
		t.Fatalf("haversine error %.4f d=%.1f", rel, d)
	}
}

func TestValidCoord(t *testing.T) {
	if ValidCoord(91, 0) || ValidCoord(0, 181) || !ValidCoord(30, 120) {
		t.Fatal("coord validation")
	}
}
