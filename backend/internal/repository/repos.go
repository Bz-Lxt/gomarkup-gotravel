package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gotravel/internal/model"
	"gotravel/internal/timeutil"
)

type Repos struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repos { return &Repos{Pool: pool} }

func (r *Repos) CreateUser(ctx context.Context, u *model.User) error {
	return r.Pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, nickname, avatar_color, created_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		u.Username, u.Password, u.Nickname, u.AvatarColor, timeutil.Now(),
	).Scan(&u.ID)
}

func (r *Repos) UserByUsername(ctx context.Context, name string) (*model.User, error) {
	u := &model.User{}
	err := r.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, nickname, avatar_color, created_at FROM users WHERE username=$1`, name,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.AvatarColor, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *Repos) UserByID(ctx context.Context, id int64) (*model.User, error) {
	u := &model.User{}
	err := r.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, nickname, avatar_color, created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Nickname, &u.AvatarColor, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *Repos) CreateTeam(ctx context.Context, t *model.Team) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	now := timeutil.Now()
	if err := tx.QueryRow(ctx,
		`INSERT INTO teams (name, leader_id, invite_code, status, created_at) VALUES ($1,$2,$3,'active',$4) RETURNING id`,
		t.Name, t.LeaderID, t.InviteCode, now,
	).Scan(&t.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role, joined_at) VALUES ($1,$2,'leader',$3)`,
		t.ID, t.LeaderID, now,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repos) TeamByID(ctx context.Context, id int64) (*model.Team, error) {
	t := &model.Team{}
	err := r.Pool.QueryRow(ctx,
		`SELECT id, name, leader_id, invite_code, status, created_at FROM teams WHERE id=$1`, id,
	).Scan(&t.ID, &t.Name, &t.LeaderID, &t.InviteCode, &t.Status, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *Repos) TeamByInvite(ctx context.Context, code string) (*model.Team, error) {
	t := &model.Team{}
	err := r.Pool.QueryRow(ctx,
		`SELECT id, name, leader_id, invite_code, status, created_at FROM teams WHERE invite_code=$1`, code,
	).Scan(&t.ID, &t.Name, &t.LeaderID, &t.InviteCode, &t.Status, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *Repos) TeamsOfUser(ctx context.Context, userID int64) ([]model.Team, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT t.id, t.name, t.leader_id, t.invite_code, t.status, t.created_at, m.role
		FROM teams t JOIN team_members m ON m.team_id=t.id
		WHERE m.user_id=$1 ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Team, 0)
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.LeaderID, &t.InviteCode, &t.Status, &t.CreatedAt, &t.Role); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repos) JoinTeam(ctx context.Context, teamID, userID int64) error {
	_, err := r.Pool.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role, joined_at) VALUES ($1,$2,'member',$3)
		 ON CONFLICT DO NOTHING`, teamID, userID, timeutil.Now())
	return err
}

func (r *Repos) MemberRole(ctx context.Context, teamID, userID int64) (string, error) {
	var role string
	err := r.Pool.QueryRow(ctx,
		`SELECT role FROM team_members WHERE team_id=$1 AND user_id=$2`, teamID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
}

func (r *Repos) Members(ctx context.Context, teamID int64) ([]model.Member, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT m.user_id, u.username, u.nickname, u.avatar_color, m.role, m.joined_at
		FROM team_members m JOIN users u ON u.id=m.user_id
		WHERE m.team_id=$1 ORDER BY m.joined_at`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Member, 0)
	for rows.Next() {
		var m model.Member
		if err := rows.Scan(&m.UserID, &m.Username, &m.Nickname, &m.AvatarColor, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repos) CreateTrip(ctx context.Context, t *model.Trip) error {
	now := timeutil.Now()
	return r.Pool.QueryRow(ctx, `
		INSERT INTO trips (team_id, title, description, cover, total_distance_m, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,0,'draft',$5,$6) RETURNING id`,
		t.TeamID, t.Title, t.Description, t.Cover, now, now,
	).Scan(&t.ID)
}

func (r *Repos) TripByID(ctx context.Context, id int64) (*model.Trip, error) {
	t := &model.Trip{}
	err := r.Pool.QueryRow(ctx, `
		SELECT id, team_id, title, description, cover, total_distance_m, status, created_at, updated_at
		FROM trips WHERE id=$1`, id,
	).Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Cover, &t.TotalDistanceM, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *Repos) TripsByTeam(ctx context.Context, teamID int64) ([]model.Trip, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, team_id, title, description, cover, total_distance_m, status, created_at, updated_at
		FROM trips WHERE team_id=$1 ORDER BY updated_at DESC`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Trip, 0)
	for rows.Next() {
		var t model.Trip
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Cover, &t.TotalDistanceM, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repos) UpdateTrip(ctx context.Context, t *model.Trip) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE trips SET title=$1, description=$2, cover=$3, total_distance_m=$4, status=$5, updated_at=$6 WHERE id=$7`,
		t.Title, t.Description, t.Cover, t.TotalDistanceM, t.Status, timeutil.Now(), t.ID)
	return err
}

func (r *Repos) DeleteTrip(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM trips WHERE id=$1`, id)
	return err
}

func (r *Repos) NextSeq(ctx context.Context, tripID int64) (int, error) {
	var n *int
	if err := r.Pool.QueryRow(ctx, `SELECT MAX(seq) FROM waypoints WHERE trip_id=$1`, tripID).Scan(&n); err != nil {
		return 0, err
	}
	if n == nil {
		return 1, nil
	}
	return *n + 1, nil
}

func (r *Repos) InsertWaypoint(ctx context.Context, w *model.Waypoint) error {
	return r.Pool.QueryRow(ctx, `
		INSERT INTO waypoints (trip_id, seq, name, kind, lat, lng, altitude, planned_stay_min, note, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		w.TripID, w.Seq, w.Name, w.Kind, w.Lat, w.Lng, w.Altitude, w.PlannedStayMin, w.Note, timeutil.Now(),
	).Scan(&w.ID)
}

func (r *Repos) Waypoints(ctx context.Context, tripID int64) ([]model.Waypoint, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, trip_id, seq, name, kind, lat, lng, altitude, planned_stay_min, note, created_at
		FROM waypoints WHERE trip_id=$1 ORDER BY seq`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Waypoint, 0)
	for rows.Next() {
		var w model.Waypoint
		if err := rows.Scan(&w.ID, &w.TripID, &w.Seq, &w.Name, &w.Kind, &w.Lat, &w.Lng, &w.Altitude, &w.PlannedStayMin, &w.Note, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repos) WaypointByID(ctx context.Context, id int64) (*model.Waypoint, error) {
	w := &model.Waypoint{}
	err := r.Pool.QueryRow(ctx, `
		SELECT id, trip_id, seq, name, kind, lat, lng, altitude, planned_stay_min, note, created_at
		FROM waypoints WHERE id=$1`, id,
	).Scan(&w.ID, &w.TripID, &w.Seq, &w.Name, &w.Kind, &w.Lat, &w.Lng, &w.Altitude, &w.PlannedStayMin, &w.Note, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

func (r *Repos) UpdateWaypoint(ctx context.Context, w *model.Waypoint) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE waypoints SET name=$1, kind=$2, lat=$3, lng=$4, altitude=$5, planned_stay_min=$6, note=$7 WHERE id=$8`,
		w.Name, w.Kind, w.Lat, w.Lng, w.Altitude, w.PlannedStayMin, w.Note, w.ID)
	return err
}

func (r *Repos) DeleteWaypoint(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM waypoints WHERE id=$1`, id)
	return err
}

func (r *Repos) ReorderWaypoints(ctx context.Context, tripID int64, ids []int64) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `UPDATE waypoints SET seq=$1 WHERE id=$2 AND trip_id=$3`, i+1, id, tripID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("waypoint %d not in trip", id)
		}
	}
	return tx.Commit(ctx)
}

func (r *Repos) AddDep(ctx context.Context, d model.Dep) error {
	_, err := r.Pool.Exec(ctx,
		`INSERT INTO waypoint_deps (trip_id, from_waypoint_id, to_waypoint_id) VALUES ($1,$2,$3)`,
		d.TripID, d.FromID, d.ToID)
	return err
}

func (r *Repos) DeleteDep(ctx context.Context, tripID, from, to int64) error {
	_, err := r.Pool.Exec(ctx,
		`DELETE FROM waypoint_deps WHERE trip_id=$1 AND from_waypoint_id=$2 AND to_waypoint_id=$3`,
		tripID, from, to)
	return err
}

func (r *Repos) Deps(ctx context.Context, tripID int64) ([]model.Dep, error) {
	rows, err := r.Pool.Query(ctx,
		`SELECT trip_id, from_waypoint_id, to_waypoint_id FROM waypoint_deps WHERE trip_id=$1`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Dep, 0)
	for rows.Next() {
		var d model.Dep
		if err := rows.Scan(&d.TripID, &d.FromID, &d.ToID); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repos) CreateSession(ctx context.Context, s *model.Session) error {
	return r.Pool.QueryRow(ctx,
		`INSERT INTO trip_sessions (trip_id, started_at, status) VALUES ($1,$2,'live') RETURNING id`,
		s.TripID, timeutil.Now(),
	).Scan(&s.ID)
}

func (r *Repos) SessionByID(ctx context.Context, id int64) (*model.Session, error) {
	s := &model.Session{}
	var ended *time.Time
	err := r.Pool.QueryRow(ctx,
		`SELECT id, trip_id, started_at, ended_at, status FROM trip_sessions WHERE id=$1`, id,
	).Scan(&s.ID, &s.TripID, &s.StartedAt, &ended, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	s.EndedAt = ended
	return s, err
}

func (r *Repos) LiveSessionByTrip(ctx context.Context, tripID int64) (*model.Session, error) {
	s := &model.Session{}
	var ended *time.Time
	err := r.Pool.QueryRow(ctx,
		`SELECT id, trip_id, started_at, ended_at, status FROM trip_sessions WHERE trip_id=$1 AND status='live' ORDER BY started_at DESC LIMIT 1`,
		tripID,
	).Scan(&s.ID, &s.TripID, &s.StartedAt, &ended, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	s.EndedAt = ended
	return s, err
}

func (r *Repos) EndSession(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `UPDATE trip_sessions SET status='ended', ended_at=$1 WHERE id=$2`, timeutil.Now(), id)
	return err
}

func (r *Repos) InsertPosition(ctx context.Context, p model.Position) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO position_reports (session_id, user_id, lat, lng, speed, accuracy, heading, reported_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.SessionID, p.UserID, p.Lat, p.Lng, p.Speed, p.Accuracy, p.Heading, p.ReportedAt)
	return err
}

func (r *Repos) RecentPositions(ctx context.Context, sessionID int64, limit int) ([]model.Position, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT id, session_id, user_id, lat, lng, speed, accuracy, heading, reported_at
		FROM position_reports WHERE session_id=$1 ORDER BY reported_at DESC LIMIT $2`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Position, 0)
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.ID, &p.SessionID, &p.UserID, &p.Lat, &p.Lng, &p.Speed, &p.Accuracy, &p.Heading, &p.ReportedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repos) InsertPhoto(ctx context.Context, p *model.Photo) error {
	return r.Pool.QueryRow(ctx, `
		INSERT INTO photos (trip_id, session_id, user_id, lat, lng, file_path, thumb_path, caption, taken_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		p.TripID, p.SessionID, p.UserID, p.Lat, p.Lng, p.FilePath, p.ThumbPath, p.Caption, p.TakenAt, timeutil.Now(),
	).Scan(&p.ID)
}

func (r *Repos) Photos(ctx context.Context, tripID int64) ([]model.Photo, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT p.id, p.trip_id, p.session_id, p.user_id, u.nickname, p.lat, p.lng, p.file_path, p.thumb_path, p.caption, p.taken_at, p.created_at
		FROM photos p JOIN users u ON u.id=p.user_id
		WHERE p.trip_id=$1 ORDER BY p.created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Photo, 0)
	for rows.Next() {
		var p model.Photo
		if err := rows.Scan(&p.ID, &p.TripID, &p.SessionID, &p.UserID, &p.Nickname, &p.Lat, &p.Lng, &p.FilePath, &p.ThumbPath, &p.Caption, &p.TakenAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repos) InsertAlert(ctx context.Context, a *model.Alert) error {
	if len(a.Payload) == 0 {
		a.Payload = []byte(`{}`)
	}
	if !json.Valid(a.Payload) {
		return fmt.Errorf("alert payload not json")
	}
	return r.Pool.QueryRow(ctx, `
		INSERT INTO alerts (session_id, type, payload, created_by, created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		a.SessionID, a.Type, a.Payload, a.CreatedBy, timeutil.Now(),
	).Scan(&a.ID)
}

func (r *Repos) Alerts(ctx context.Context, sessionID int64) ([]model.Alert, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT a.id, a.session_id, a.type, a.payload, a.created_by, a.created_at,
		       (SELECT COUNT(*) FROM alert_acks k WHERE k.alert_id=a.id)
		FROM alerts a WHERE a.session_id=$1 ORDER BY a.created_at DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Alert, 0)
	for rows.Next() {
		var a model.Alert
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Type, &a.Payload, &a.CreatedBy, &a.CreatedAt, &a.Acks); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repos) AckAlert(ctx context.Context, alertID, userID int64) error {
	_, err := r.Pool.Exec(ctx,
		`INSERT INTO alert_acks (alert_id, user_id, acked_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		alertID, userID, timeutil.Now())
	return err
}

func (r *Repos) PurgeOldPositions(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM position_reports WHERE reported_at < $1`, timeutil.Now().Add(-olderThan))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
