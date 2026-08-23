package ws

import (
	"sync"
	"sync/atomic"
	"time"
)

type Snapshot struct {
	Inbound        int64   `json:"inbound"`
	Outbound       int64   `json:"outbound"`
	NaiveOutbound  int64   `json:"naive_outbound"`
	CoalesceRatio  float64 `json:"coalesce_ratio"`
	Drops          int64   `json:"drops"`
	Kicks          int64   `json:"kicks"`
	Rooms          int64   `json:"rooms"`
	Clients        int64   `json:"clients"`
	RateLimited    int64   `json:"rate_limited"`
	P99BroadcastMs float64 `json:"p99_broadcast_ms"`
	GoroutinesHint string  `json:"goroutines_hint"`
}

type Metrics struct {
	Inbound     atomic.Int64
	Outbound    atomic.Int64
	Naive       atomic.Int64
	Drops       atomic.Int64
	Kicks       atomic.Int64
	RateLimited atomic.Int64
	mu          sync.Mutex
	latencies   []float64
}

func (m *Metrics) ObserveBroadcast(d time.Duration) {
	ms := float64(d.Microseconds()) / 1000
	m.mu.Lock()
	m.latencies = append(m.latencies, ms)
	if len(m.latencies) > 2048 {
		m.latencies = m.latencies[len(m.latencies)-1024:]
	}
	m.mu.Unlock()
}

func (m *Metrics) P99() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.latencies) == 0 {
		return 0
	}
	cp := append([]float64(nil), m.latencies...)
	// insertion sort
	for i := 1; i < len(cp); i++ {
		j := i
		for j > 0 && cp[j] < cp[j-1] {
			cp[j], cp[j-1] = cp[j-1], cp[j]
			j--
		}
	}
	idx := int(float64(len(cp)-1) * 0.99)
	return cp[idx]
}

func (m *Metrics) Snapshot(rooms, clients int64) Snapshot {
	in := m.Inbound.Load()
	out := m.Outbound.Load()
	naive := m.Naive.Load()
	ratio := 0.0
	if naive > 0 {
		ratio = 1 - float64(out)/float64(naive)
		if ratio < 0 {
			ratio = 0
		}
	}
	return Snapshot{
		Inbound:        in,
		Outbound:       out,
		NaiveOutbound:  naive,
		CoalesceRatio:  ratio,
		Drops:          m.Drops.Load(),
		Kicks:          m.Kicks.Load(),
		Rooms:          rooms,
		Clients:        clients,
		RateLimited:    m.RateLimited.Load(),
		P99BroadcastMs: m.P99(),
	}
}
