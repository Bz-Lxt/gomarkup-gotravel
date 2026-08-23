package ws

import "encoding/json"

const (
	TypePos      = "pos"
	TypeBatchPos = "batch_pos"
	TypeRally    = "rally"
	TypeLaggard  = "laggard"
	TypePresence = "presence"
	TypePing     = "ping"
	TypePong     = "pong"
	TypeAck      = "ack"
	TypeError    = "error"
	TypeWelcome  = "welcome"
	TypeRate     = "RATE_LIMITED"
)

type Envelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

type PosIn struct {
	Type     string  `json:"type"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Speed    float64 `json:"speed"`
	Accuracy float64 `json:"accuracy"`
	Heading  float64 `json:"heading"`
	TS       int64   `json:"ts"`
}

type MemberPos struct {
	UserID      int64   `json:"user_id"`
	Nickname    string  `json:"nickname"`
	AvatarColor string  `json:"avatar_color"`
	Role        string  `json:"role"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Speed       float64 `json:"speed"`
	Heading     float64 `json:"heading"`
	Net         string  `json:"net"`
	TS          int64   `json:"ts"`
}

type BatchPos struct {
	Type    string      `json:"type"`
	Members []MemberPos `json:"members"`
	Tick    int64       `json:"tick"`
}

type RallyOut struct {
	Type      string  `json:"type"`
	AlertID   int64   `json:"alert_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Message   string  `json:"message"`
	CreatedAt string  `json:"created_at"`
}

type LaggardOut struct {
	Type      string  `json:"type"`
	AlertID   int64   `json:"alert_id"`
	UserID    int64   `json:"user_id"`
	Nickname  string  `json:"nickname"`
	DistanceM float64 `json:"distance_m"`
	Mode      string  `json:"mode"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
}

type AckIn struct {
	Type    string `json:"type"`
	AlertID int64  `json:"alert_id"`
}

type ErrOut struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Welcome struct {
	Type      string `json:"type"`
	SessionID int64  `json:"session_id"`
	SelfID    int64  `json:"self_id"`
}

func MustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","code":"INTERNAL","message":"marshal"}`)
	}
	return b
}
