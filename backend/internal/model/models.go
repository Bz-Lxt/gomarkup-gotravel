package model

import "time"

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"-"`
	Nickname    string    `json:"nickname"`
	AvatarColor string    `json:"avatar_color"`
	CreatedAt   time.Time `json:"created_at"`
}

type Team struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	LeaderID   int64     `json:"leader_id"`
	InviteCode string    `json:"invite_code"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Role       string    `json:"role,omitempty"`
}

type Member struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	AvatarColor string    `json:"avatar_color"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

type Trip struct {
	ID             int64     `json:"id"`
	TeamID         int64     `json:"team_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Cover          string    `json:"cover"`
	TotalDistanceM float64   `json:"total_distance_m"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	KindStop    = "STOP"
	KindCheckin = "CHECKIN"
	KindLodging = "LODGING"
)

type Waypoint struct {
	ID             int64     `json:"id"`
	TripID         int64     `json:"trip_id"`
	Seq            int       `json:"seq"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	Lat            float64   `json:"lat"`
	Lng            float64   `json:"lng"`
	Altitude       float64   `json:"altitude"`
	PlannedStayMin int       `json:"planned_stay_min"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"created_at"`
}

type Dep struct {
	TripID int64 `json:"trip_id"`
	FromID int64 `json:"from_waypoint_id"`
	ToID   int64 `json:"to_waypoint_id"`
}

type Session struct {
	ID        int64      `json:"id"`
	TripID    int64      `json:"trip_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Status    string     `json:"status"`
}

type Photo struct {
	ID        int64     `json:"id"`
	TripID    int64     `json:"trip_id"`
	SessionID *int64    `json:"session_id,omitempty"`
	UserID    int64     `json:"user_id"`
	Nickname  string    `json:"nickname,omitempty"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	URL       string    `json:"url"`
	ThumbURL  string    `json:"thumb_url"`
	FilePath  string    `json:"-"`
	ThumbPath string    `json:"-"`
	Caption   string    `json:"caption"`
	TakenAt   time.Time `json:"taken_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Alert struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"session_id"`
	Type      string    `json:"type"`
	Payload   []byte    `json:"payload"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Acks      int       `json:"acks"`
}

type Position struct {
	ID         int64     `json:"id"`
	SessionID  int64     `json:"session_id"`
	UserID     int64     `json:"user_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Speed      float64   `json:"speed"`
	Accuracy   float64   `json:"accuracy"`
	Heading    float64   `json:"heading"`
	ReportedAt time.Time `json:"reported_at"`
}
