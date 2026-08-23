package geo

import (
	"context"
	"sync"
	"time"

	"gotravel/internal/timeutil"
)

type Mode string

const (
	ModeCentroid Mode = "centroid"
	ModeLeader   Mode = "leader"
	ModeNextWP   Mode = "next_waypoint"
)

type Alert struct {
	UserID     int64
	DistanceM  float64
	Mode       Mode
	Lat, Lng   float64
	RefLat     float64
	RefLng     float64
	At         time.Time
}

type Engine struct {
	idx       Index
	threshold float64
	cooldown  time.Duration
	mu        sync.Mutex
	lastFire  map[string]time.Time // session:user
}

func NewEngine(idx Index, thresholdM float64, cooldown time.Duration) *Engine {
	if thresholdM <= 0 {
		thresholdM = 2000
	}
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	return &Engine{idx: idx, threshold: thresholdM, cooldown: cooldown, lastFire: make(map[string]time.Time)}
}

func (e *Engine) Index() Index { return e.idx }

func (e *Engine) Evaluate(ctx context.Context, sessionKey string, leaderID int64, nextWP *[2]float64) (*Alert, error) {
	members, err := e.idx.All(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if len(members) < 2 {
		return nil, nil
	}
	pts := make([][2]float64, 0, len(members))
	var leader *Member
	for i := range members {
		pts = append(pts, [2]float64{members[i].Lat, members[i].Lng})
		if id, err := ParseMemberID(members[i].ID); err == nil && id == leaderID {
			cp := members[i]
			leader = &cp
		}
	}
	cLat, cLng := Centroid(pts)
	mode := ModeCentroid
	refLat, refLng := cLat, cLng
	if nextWP != nil {
		mode = ModeNextWP
		refLat, refLng = nextWP[0], nextWP[1]
	} else if leader != nil {
		mode = ModeLeader
		refLat, refLng = leader.Lat, leader.Lng
	}
	far, dist, err := e.idx.Farthest(ctx, sessionKey, refLat, refLng)
	if err != nil {
		return nil, err
	}
	if dist < e.threshold {
		return nil, nil
	}
	uid, err := ParseMemberID(far.ID)
	if err != nil {
		return nil, err
	}
	ck := sessionKey + ":" + far.ID
	e.mu.Lock()
	defer e.mu.Unlock()
	if t, ok := e.lastFire[ck]; ok && timeutil.Now().Sub(t) < e.cooldown {
		return nil, nil
	}
	e.lastFire[ck] = timeutil.Now()
	return &Alert{
		UserID:    uid,
		DistanceM: dist,
		Mode:      mode,
		Lat:       far.Lat,
		Lng:       far.Lng,
		RefLat:    refLat,
		RefLng:    refLng,
		At:        timeutil.Now(),
	}, nil
}

func FarthestBrute(members []Member, lat, lng float64) (Member, float64) {
	var best Member
	bestD := -1.0
	for _, m := range members {
		d := Haversine(lat, lng, m.Lat, m.Lng)
		if d > bestD {
			bestD = d
			best = m
		}
	}
	return best, bestD
}
