package ws

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendBuf     = 64
	prioBuf     = 32
	writeWait   = 2 * time.Second
	pongWait    = 30 * time.Second
	pingPeriod  = 20 * time.Second
	maxSlowFail = 3
	maxDropRun  = 5
)

type Client struct {
	UserID      int64
	Nickname    string
	AvatarColor string
	Role        string
	SessionID   int64
	conn        *websocket.Conn
	send        chan []byte
	prio        chan []byte
	bucket      *TokenBucket
	hub         *Hub
	slowFails   atomic.Int32
	dropRun     atomic.Int32
	closed      atomic.Bool
	LastSeen    atomic.Int64
}

func newClient(h *Hub, conn *websocket.Conn, sessionID, userID int64, nick, color, role string) *Client {
	c := &Client{
		UserID:      userID,
		Nickname:    nick,
		AvatarColor: color,
		Role:        role,
		SessionID:   sessionID,
		conn:        conn,
		send:        make(chan []byte, sendBuf),
		prio:        make(chan []byte, prioBuf),
		bucket:      NewTokenBucket(5, 10),
		hub:         h,
	}
	c.LastSeen.Store(time.Now().UnixMilli())
	return c
}

func (c *Client) Close() {
	if c.closed.CompareAndSwap(false, true) {
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}
}

// EnqueuePos drop-oldest on overflow; never blocks Hub.
func (c *Client) EnqueuePos(payload []byte) {
	select {
	case c.send <- payload:
		c.dropRun.Store(0)
		return
	default:
	}
	select {
	case <-c.send:
		c.hub.Metrics.Drops.Add(1)
		n := c.dropRun.Add(1)
		if n > maxDropRun {
			c.hub.kick(c, "drop_run")
			return
		}
	default:
	}
	select {
	case c.send <- payload:
	default:
		c.hub.Metrics.Drops.Add(1)
	}
}

func (c *Client) EnqueuePrio(payload []byte) {
	select {
	case c.prio <- payload:
		return
	default:
	}
	// priority channel must not drop: try a short block then kick
	timer := time.NewTimer(150 * time.Millisecond)
	select {
	case c.prio <- payload:
		timer.Stop()
	case <-timer.C:
		c.hub.kick(c, "prio_backpressure")
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
		c.hub.unregister(c)
	}()
	for {
		select {
		case msg, ok := <-c.prio:
			if !ok {
				return
			}
			if !c.writeRaw(websocket.TextMessage, msg) {
				return
			}
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if !c.writeRaw(websocket.TextMessage, msg) {
				return
			}
		case <-ticker.C:
			if !c.writeRaw(websocket.PingMessage, nil) {
				return
			}
		}
	}
}

func (c *Client) writeRaw(mt int, payload []byte) bool {
	if c.conn == nil {
		return true
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(mt, payload); err != nil {
		n := c.slowFails.Add(1)
		if n >= maxSlowFail {
			c.hub.Metrics.Kicks.Add(1)
			return false
		}
		return true
	}
	c.slowFails.Store(0)
	if mt == websocket.TextMessage {
		c.hub.Metrics.Outbound.Add(1)
	}
	return true
}
