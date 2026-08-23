package geo

import (
	"context"
	"testing"
	"time"
)

func TestLaggardCooldown(t *testing.T) {
	idx := NewRTreeIndex()
	ctx := context.Background()
	_ = idx.Upsert(ctx, "s", Member{ID: "1", Lat: 30.25, Lng: 120.15})
	_ = idx.Upsert(ctx, "s", Member{ID: "2", Lat: 30.28, Lng: 120.18})
	_ = idx.Upsert(ctx, "s", Member{ID: "3", Lat: 30.40, Lng: 120.40})
	e := NewEngine(idx, 500, 60*time.Second)
	a1, err := e.Evaluate(ctx, "s", 1, nil)
	if err != nil || a1 == nil {
		t.Fatalf("expected alert %v %v", a1, err)
	}
	a2, err := e.Evaluate(ctx, "s", 1, nil)
	if err != nil || a2 != nil {
		t.Fatalf("cooldown failed %+v", a2)
	}
}
