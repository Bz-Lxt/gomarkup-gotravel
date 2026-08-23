package ws_test

import (
	"context"
	"testing"
	"time"

	"gotravel/internal/geo"
	"gotravel/internal/ws"
)

func TestHubInjectPosInitializesDefaultGeoIndex(t *testing.T) {
	hub := ws.NewHub()
	defer hub.Stop()
	hub.Geo = geo.NewEngine(geo.NewRTreeIndex(), 2000, time.Minute)

	hub.InjectPos(42, ws.MemberPos{UserID: 7, Lat: 31.2304, Lng: 121.4737})

	members, err := hub.Geo.Index().All(context.Background(), "42")
	if err != nil {
		t.Fatalf("read indexed positions: %v", err)
	}
	if len(members) != 1 || members[0].ID != "7" {
		t.Fatalf("indexed members = %+v, want user 7", members)
	}
}
