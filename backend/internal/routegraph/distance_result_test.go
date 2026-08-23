package routegraph_test

import (
	"reflect"
	"testing"

	"gotravel/internal/routegraph"
)

func TestHaversineResultsRemainStableAcrossCalls(t *testing.T) {
	provider := routegraph.HaversineProvider{}
	first, err := provider.Compute([]routegraph.Point{
		{Lat: 39.9042, Lng: 116.4074},
		{Lat: 39.9142, Lng: 116.4174},
		{Lat: 39.9242, Lng: 116.4274},
	})
	if err != nil {
		t.Fatalf("compute first route: %v", err)
	}
	wantSegments := append([]routegraph.Segment(nil), first.Segments...)

	second, err := provider.Compute([]routegraph.Point{
		{Lat: 31.2304, Lng: 121.4737},
		{Lat: 31.3304, Lng: 121.5737},
		{Lat: 31.4304, Lng: 121.6737},
	})
	if err != nil {
		t.Fatalf("compute second route: %v", err)
	}
	if reflect.DeepEqual(second.Segments, wantSegments) {
		t.Fatal("test routes unexpectedly produced identical segments")
	}
	if !reflect.DeepEqual(first.Segments, wantSegments) {
		t.Fatalf("first result changed after another computation: got %+v, want %+v", first.Segments, wantSegments)
	}
}
