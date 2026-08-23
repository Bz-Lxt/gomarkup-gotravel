package ws

import (
	"sync"
	"time"
)

// Batcher coalesces inbound position frames: keep latest per user, emit every tick.
type Batcher struct {
	tick time.Duration
	mu   sync.Mutex
	buf  map[int64]MemberPos
}

func NewBatcher(tick time.Duration) *Batcher {
	if tick <= 0 {
		tick = 200 * time.Millisecond
	}
	return &Batcher{tick: tick, buf: make(map[int64]MemberPos)}
}

func (b *Batcher) Tick() time.Duration { return b.tick }

func (b *Batcher) Put(p MemberPos) {
	b.mu.Lock()
	b.buf[p.UserID] = p
	b.mu.Unlock()
}

func (b *Batcher) Flush() []MemberPos {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf) == 0 {
		return nil
	}
	out := make([]MemberPos, 0, len(b.buf))
	for _, p := range b.buf {
		out = append(out, p)
	}
	b.buf = make(map[int64]MemberPos)
	return out
}

func (b *Batcher) Pending() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buf)
}
