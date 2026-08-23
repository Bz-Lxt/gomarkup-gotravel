package routegraph

import "testing"

// TestComputeNoCrossContamination locks down the "串单" regression: computing
// route B after route A must not mutate A's already-returned segments.
func TestComputeNoCrossContamination(t *testing.T) {
	a := []Point{{Lat: 30.0, Lng: 120.0}, {Lat: 30.1, Lng: 120.1}, {Lat: 30.2, Lng: 120.2}}
	b := []Point{{Lat: 40.0, Lng: 116.0}, {Lat: 40.5, Lng: 116.5}, {Lat: 41.0, Lng: 117.0}, {Lat: 41.5, Lng: 117.5}}

	p := HaversineProvider{}
	resA, err := p.Compute(a)
	if err != nil {
		t.Fatalf("A: %v", err)
	}
	wantA := append([]Segment{}, resA.Segments...) // snapshot A's segments
	resB, err := p.Compute(b)
	if err != nil {
		t.Fatalf("B: %v", err)
	}

	// A's total is a value scalar; verify it didn't drift.
	if resA.TotalMeters == 0 {
		t.Fatal("A total zero")
	}
	// A's segments must be byte-for-byte the same as right after A was computed.
	if len(resA.Segments) != len(wantA) {
		t.Fatalf("A segs len %d -> %d after B", len(wantA), len(resA.Segments))
	}
	for i := range wantA {
		if resA.Segments[i] != wantA[i] {
			t.Fatalf("A seg[%d] mutated by B: %+v != %+v", i, resA.Segments[i], wantA[i])
		}
	}
	// B must reflect its own (different) indices/meters.
	if len(resB.Segments) != len(b)-1 {
		t.Fatalf("B segs len %d want %d", len(resB.Segments), len(b)-1)
	}
	if resB.Segments[len(resB.Segments)-1].ToIdx != len(b)-1 {
		t.Fatalf("B last seg ToIdx %d want %d", resB.Segments[len(resB.Segments)-1].ToIdx, len(b)-1)
	}
	// Backing arrays must be independent (no aliasing).
	if cap(resA.Segments) > 0 && len(resB.Segments) > 0 && &resA.Segments[:1][0] == &resB.Segments[:1][0] {
		t.Fatal("A and B segments alias the same backing array")
	}
}
