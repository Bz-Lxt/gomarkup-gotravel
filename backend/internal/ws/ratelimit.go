package ws

import (
	"sync"
	"time"
)

// TokenBucket is a per-connection inbound limiter (default 5/s, burst 10).
type TokenBucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	mu     sync.Mutex
}

func NewTokenBucket(rate, burst float64) *TokenBucket {
	if rate <= 0 {
		rate = 5
	}
	if burst <= 0 {
		burst = 10
	}
	return &TokenBucket{rate: rate, burst: burst, tokens: burst, last: time.Now()}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
