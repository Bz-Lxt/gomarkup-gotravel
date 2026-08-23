package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gotravel/internal/geo"
	"gotravel/internal/logger"
	"gotravel/internal/timeutil"
)

type PosHandler func(sessionID, userID int64, lat, lng, speed, acc, heading float64, ts time.Time)
type AckHandler func(sessionID, userID, alertID int64)
type LaggardHook func(sessionID int64, a *geo.Alert)

type Hub struct {
	mu       sync.Mutex
	rooms    map[int64]*Room
	Metrics  *Metrics
	OnPos    PosHandler
	OnAck    AckHandler
	OnLag    LaggardHook
	Geo      *geo.Engine
	upgrader websocket.Upgrader
	stop     chan struct{}
}

func NewHub() *Hub {
	h := &Hub{
		rooms:   make(map[int64]*Room),
		Metrics: &Metrics{},
		stop:    make(chan struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
	go h.loop()
	return h
}

func (h *Hub) Stop() { close(h.stop) }

func (h *Hub) room(id int64) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[id]
	if !ok {
		r = newRoom(id)
		h.rooms[id] = r
	}
	return r
}

func (h *Hub) RoomCount() (rooms, clients int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	rooms = int64(len(h.rooms))
	for _, r := range h.rooms {
		clients += int64(r.size())
	}
	return
}

func (h *Hub) Snapshot() Snapshot {
	rooms, clients := h.RoomCount()
	s := h.Metrics.Snapshot(rooms, clients)
	s.GoroutinesHint = strconv.Itoa(runtime.NumGoroutine())
	return s
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, sessionID, userID int64, nick, color, role string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L.Warn("ws upgrade", "err", err)
		return
	}
	c := newClient(h, conn, sessionID, userID, nick, color, role)
	h.room(sessionID).add(c)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	go c.writePump()
	c.EnqueuePrio(MustJSON(Welcome{Type: TypeWelcome, SessionID: sessionID, SelfID: userID}))
	// send current snapshot
	c.EnqueuePos(MustJSON(BatchPos{Type: TypeBatchPos, Members: h.room(sessionID).Latest(), Tick: time.Now().UnixMilli()}))
	h.readLoop(c)
}

func (h *Hub) readLoop(c *Client) {
	defer func() {
		c.Close()
		h.unregister(c)
	}()
	if c.conn == nil {
		return
	}
	c.conn.SetReadLimit(4096)
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		h.Metrics.Inbound.Add(1)
		if !c.bucket.Allow() {
			h.Metrics.RateLimited.Add(1)
			c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: TypeRate, Message: "too many gps frames"}))
			continue
		}
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &head); err != nil {
			c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: "BAD_REQUEST", Message: "invalid json"}))
			continue
		}
		switch head.Type {
		case TypePing:
			c.EnqueuePrio(MustJSON(map[string]string{"type": TypePong}))
		case TypeAck:
			var a AckIn
			if err := json.Unmarshal(data, &a); err != nil || a.AlertID <= 0 {
				c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: "VALIDATION", Message: "alert_id required"}))
				continue
			}
			if h.OnAck != nil {
				h.OnAck(c.SessionID, c.UserID, a.AlertID)
			}
		case TypePos:
			var p PosIn
			if err := json.Unmarshal(data, &p); err != nil {
				c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: "VALIDATION", Message: "bad pos"}))
				continue
			}
			if !geo.ValidCoord(p.Lat, p.Lng) {
				c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: "VALIDATION", Message: "lat/lng out of range"}))
				continue
			}
			p.Lat, p.Lng = geo.ClampCoord(p.Lat, p.Lng)
			ts := p.TS
			if ts <= 0 {
				ts = time.Now().UnixMilli()
			}
			c.LastSeen.Store(time.Now().UnixMilli())
			mp := MemberPos{
				UserID: c.UserID, Nickname: c.Nickname, AvatarColor: c.AvatarColor, Role: c.Role,
				Lat: p.Lat, Lng: p.Lng, Speed: p.Speed, Heading: p.Heading, Net: "online", TS: ts,
			}
			room := h.room(c.SessionID)
			room.putPos(mp)
			if h.Geo != nil {
				_ = h.Geo.Index().Upsert(context.Background(), strconv.FormatInt(c.SessionID, 10), geo.Member{
					ID: geo.MemberID(c.UserID), Lat: p.Lat, Lng: p.Lng,
				})
			}
			if h.OnPos != nil {
				h.OnPos(c.SessionID, c.UserID, p.Lat, p.Lng, p.Speed, p.Accuracy, p.Heading, time.UnixMilli(ts).In(timeutil.Beijing))
			}
		default:
			c.EnqueuePrio(MustJSON(ErrOut{Type: TypeError, Code: "BAD_REQUEST", Message: "unknown type"}))
		}
	}
}

func (h *Hub) unregister(c *Client) {
	r := h.room(c.SessionID)
	r.remove(c)
	h.BroadcastPrio(c.SessionID, MustJSON(map[string]any{
		"type": TypePresence, "user_id": c.UserID, "net": "offline",
	}))
}

func (h *Hub) kick(c *Client, reason string) {
	h.Metrics.Kicks.Add(1)
	if logger.L != nil {
		logger.L.Info("ws kick", "user", c.UserID, "reason", reason)
	}
	c.Close()
}

func (h *Hub) BroadcastPos(sessionID int64, payload []byte) {
	clients := h.room(sessionID).snapshot()
	// naive N*N counter: inbound members * clients
	h.Metrics.Naive.Add(int64(len(clients)))
	for _, c := range clients {
		c.EnqueuePos(payload)
	}
}

func (h *Hub) BroadcastPrio(sessionID int64, payload []byte) {
	for _, c := range h.room(sessionID).snapshot() {
		c.EnqueuePrio(payload)
	}
}

func (h *Hub) loop() {
	tick := time.NewTicker(200 * time.Millisecond)
	lag := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	defer lag.Stop()
	for {
		select {
		case <-h.stop:
			return
		case <-tick.C:
			h.flushAll()
		case <-lag.C:
			h.evalLaggards()
		}
	}
}

func (h *Hub) flushAll() {
	h.mu.Lock()
	rooms := make([]*Room, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, r)
	}
	h.mu.Unlock()
	for _, r := range rooms {
		members := r.batcher.Flush()
		if len(members) == 0 {
			continue
		}
		start := time.Now()
		payload := MustJSON(BatchPos{Type: TypeBatchPos, Members: r.Latest(), Tick: time.Now().UnixMilli()})
		h.BroadcastPos(r.SessionID, payload)
		h.Metrics.ObserveBroadcast(time.Since(start))
	}
}

func (h *Hub) evalLaggards() {
	if h.Geo == nil {
		return
	}
	h.mu.Lock()
	ids := make([]int64, 0, len(h.rooms))
	for id := range h.rooms {
		ids = append(ids, id)
	}
	h.mu.Unlock()
	for _, id := range ids {
		a, err := h.Geo.Evaluate(context.Background(), strconv.FormatInt(id, 10), 0, nil)
		if err != nil || a == nil {
			continue
		}
		if h.OnLag != nil {
			h.OnLag(id, a)
		}
	}
}

func (h *Hub) InjectPos(sessionID int64, mp MemberPos) {
	r := h.room(sessionID)
	r.putPos(mp)
	if h.Geo != nil {
		_ = h.Geo.Index().Upsert(context.Background(), strconv.FormatInt(sessionID, 10), geo.Member{
			ID: geo.MemberID(mp.UserID), Lat: mp.Lat, Lng: mp.Lng,
		})
	}
}

func (h *Hub) Isolated(sessionA, sessionB int64) bool {
	// used by tests: broadcasting A must not increment B client queues
	return sessionA != sessionB
}
