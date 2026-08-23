package ws

import (
	"testing"
	"time"
)

func TestBatcherKeepsLatest(t *testing.T) {
	b := NewBatcher(200 * time.Millisecond)
	b.Put(MemberPos{UserID: 1, Lat: 1})
	b.Put(MemberPos{UserID: 1, Lat: 2})
	b.Put(MemberPos{UserID: 2, Lat: 3})
	out := b.Flush()
	if len(out) != 2 {
		t.Fatalf("coalesce want 2 got %d", len(out))
	}
	if b.Flush() != nil {
		t.Fatal("second flush should be empty")
	}
}

func TestDropOldestDoesNotBlock(t *testing.T) {
	h := NewHub()
	defer h.Stop()
	c := newClient(h, nil, 1, 9, "n", "#000", "member")
	done := make(chan struct{})
	go func() {
		for i := 0; i < sendBuf+40; i++ {
			c.EnqueuePos([]byte(`{"type":"batch_pos"}`))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("enqueue blocked hub")
	}
	if len(c.send) > sendBuf {
		t.Fatal("buffer grew unbounded")
	}
}

func TestTokenBucket(t *testing.T) {
	b := NewTokenBucket(5, 10)
	ok := 0
	for i := 0; i < 20; i++ {
		if b.Allow() {
			ok++
		}
	}
	if ok != 10 {
		t.Fatalf("burst want 10 got %d", ok)
	}
}

func TestCompressRatio(t *testing.T) {
	if r := CompressRatio(400, 80); r < 0.79 {
		t.Fatalf("ratio %f", r)
	}
}

func TestRoomIsolation(t *testing.T) {
	h := NewHub()
	defer h.Stop()
	if !h.Isolated(1, 2) {
		t.Fatal("rooms must isolate")
	}
}
