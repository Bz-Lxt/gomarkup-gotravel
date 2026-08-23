package ws

import (
	"sync"
	"time"
)

type Room struct {
	SessionID int64
	mu        sync.RWMutex
	clients   map[int64]*Client
	batcher   *Batcher
	latest    map[int64]MemberPos
}

func newRoom(id int64) *Room {
	return &Room{
		SessionID: id,
		clients:   make(map[int64]*Client),
		batcher:   NewBatcher(200 * time.Millisecond),
		latest:    make(map[int64]MemberPos),
	}
}

func (r *Room) add(c *Client) {
	r.mu.Lock()
	r.clients[c.UserID] = c
	r.mu.Unlock()
}

func (r *Room) remove(c *Client) {
	r.mu.Lock()
	cur, ok := r.clients[c.UserID]
	if ok && cur == c {
		delete(r.clients, c.UserID)
	}
	r.mu.Unlock()
}

func (r *Room) snapshot() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		out = append(out, c)
	}
	return out
}

func (r *Room) putPos(p MemberPos) {
	r.mu.Lock()
	r.latest[p.UserID] = p
	r.mu.Unlock()
	r.batcher.Put(p)
}

func (r *Room) Latest() []MemberPos {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MemberPos, 0, len(r.latest))
	now := time.Now().UnixMilli()
	for _, p := range r.latest {
		age := now - p.TS
		switch {
		case age > 15000:
			p.Net = "offline"
		case age > 3000:
			p.Net = "weak"
		default:
			p.Net = "online"
		}
		out = append(out, p)
	}
	return out
}

func (r *Room) size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}
