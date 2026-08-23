package geo

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
)

func TestRTreeFarthestMatchesBrute(t *testing.T) {
	idx := NewRTreeIndex()
	ctx := context.Background()
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 1000; trial++ {
		key := "t" + strconv.Itoa(trial)
		n := 20 + rng.Intn(30)
		var members []Member
		for i := 0; i < n; i++ {
			m := Member{ID: strconv.Itoa(i), Lat: 30 + rng.Float64()*2, Lng: 120 + rng.Float64()*2}
			members = append(members, m)
			if err := idx.Upsert(ctx, key, m); err != nil {
				t.Fatal(err)
			}
		}
		lat, lng := 31.0, 121.0
		got, d, err := idx.Farthest(ctx, key, lat, lng)
		if err != nil {
			t.Fatal(err)
		}
		want, wd := FarthestBrute(members, lat, lng)
		if got.ID != want.ID {
			t.Fatalf("trial %d farthest id %s want %s (d=%.1f/%.1f)", trial, got.ID, want.ID, d, wd)
		}
	}
}

func TestRTreeRange(t *testing.T) {
	idx := NewRTreeIndex()
	ctx := context.Background()
	_ = idx.Upsert(ctx, "k", Member{ID: "a", Lat: 30.2, Lng: 120.1})
	_ = idx.Upsert(ctx, "k", Member{ID: "b", Lat: 31.5, Lng: 121.2})
	hits := idx.RangeSearch("k", 30, 120, 30.5, 120.5)
	if len(hits) != 1 || hits[0].ID != "a" {
		t.Fatalf("range %+v", hits)
	}
}
