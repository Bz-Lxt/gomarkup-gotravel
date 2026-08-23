package simulator

import (
	"fmt"
	"math"
	"sync"
	"time"

	"gotravel/internal/geo"
	"gotravel/internal/ws"
)

type GPS struct {
	Hub    *ws.Hub
	mu     sync.Mutex
	stop   chan struct{}
	running bool
	info   Status
}

type Status struct {
	Running   bool    `json:"running"`
	SessionID int64   `json:"session_id"`
	Count     int     `json:"count"`
	Provider  string  `json:"provider"`
	Laggard   bool    `json:"laggard"`
}

func New(hub *ws.Hub) *GPS {
	return &GPS{Hub: hub, info: Status{Provider: "sim"}}
}

func (g *GPS) Status() Status {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.info
}

func (g *GPS) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running {
		close(g.stop)
		g.running = false
		g.info.Running = false
	}
}

func (g *GPS) Start(sessionID int64, line [][2]float64, count int, withLaggard bool) error {
	if len(line) < 2 {
		return fmt.Errorf("need at least 2 waypoints to simulate")
	}
	if count <= 0 {
		count = 8
	}
	if count > 20 {
		count = 20
	}
	g.Stop()
	g.mu.Lock()
	g.stop = make(chan struct{})
	g.running = true
	g.info = Status{Running: true, SessionID: sessionID, Count: count, Provider: "sim", Laggard: withLaggard}
	stop := g.stop
	g.mu.Unlock()
	go g.loop(sessionID, line, count, withLaggard, stop)
	return nil
}

func (g *GPS) loop(sessionID int64, line [][2]float64, count int, withLaggard bool, stop chan struct{}) {
	segs := make([]float64, len(line)-1)
	var total float64
	for i := 1; i < len(line); i++ {
		segs[i-1] = geo.Haversine(line[i-1][0], line[i-1][1], line[i][0], line[i][1])
		total += segs[i-1]
	}
	if total <= 0 {
		total = 1
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	step := 0
	palette := []string{"#2F6F4E", "#C46A2B", "#1F4E79", "#8B3A3A", "#6B4C9A", "#3D7A6A"}
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			step++
			for i := 0; i < count; i++ {
				progress := math.Mod(float64(step)*0.04+float64(i)*0.03, 1)
				if withLaggard && i == count-1 {
					progress = math.Max(0, progress-0.45)
				}
				lat, lng := along(line, segs, total, progress)
				if withLaggard && i == count-1 {
					lat -= 0.03
				}
				jitter := (float64((i*17+step)%7) - 3) * 0.00008
				mp := ws.MemberPos{
					UserID:      9000 + int64(i),
					Nickname:    fmt.Sprintf("模拟队员%02d", i+1),
					AvatarColor: palette[i%len(palette)],
					Role:        "member",
					Lat:         lat + jitter,
					Lng:         lng + jitter*0.6,
					Speed:       1.2,
					Heading:     45,
					Net:         "online",
					TS:          time.Now().UnixMilli(),
				}
				if i%9 == 0 && step%8 == 0 {
					mp.Net = "weak"
				}
				g.Hub.InjectPos(sessionID, mp)
			}
		}
	}
}

func along(line [][2]float64, segs []float64, total, t float64) (float64, float64) {
	if t <= 0 {
		return line[0][0], line[0][1]
	}
	if t >= 1 {
		n := line[len(line)-1]
		return n[0], n[1]
	}
	remain := total * t
	for i, d := range segs {
		if remain <= d {
			f := 0.0
			if d > 0 {
				f = remain / d
			}
			return geo.Interpolate(line[i][0], line[i][1], line[i+1][0], line[i+1][1], f)
		}
		remain -= d
	}
	n := line[len(line)-1]
	return n[0], n[1]
}
