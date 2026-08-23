package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"gotravel/internal/logger"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(64) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  nickname VARCHAR(64) NOT NULL,
  avatar_color VARCHAR(16) NOT NULL,
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS teams (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  leader_id BIGINT NOT NULL REFERENCES users(id),
  invite_code VARCHAR(8) UNIQUE NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS team_members (
  team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(16) NOT NULL,
  joined_at TIMESTAMP NOT NULL,
  PRIMARY KEY (team_id, user_id)
);
CREATE TABLE IF NOT EXISTS trips (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  cover TEXT NOT NULL DEFAULT '',
  total_distance_m DOUBLE PRECISION NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS waypoints (
  id BIGSERIAL PRIMARY KEY,
  trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  seq INT NOT NULL,
  name VARCHAR(200) NOT NULL,
  kind VARCHAR(16) NOT NULL,
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  altitude DOUBLE PRECISION NOT NULL DEFAULT 0,
  planned_stay_min INT NOT NULL DEFAULT 0,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wp_trip_seq ON waypoints(trip_id, seq);
CREATE TABLE IF NOT EXISTS waypoint_deps (
  trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  from_waypoint_id BIGINT NOT NULL REFERENCES waypoints(id) ON DELETE CASCADE,
  to_waypoint_id BIGINT NOT NULL REFERENCES waypoints(id) ON DELETE CASCADE,
  PRIMARY KEY (from_waypoint_id, to_waypoint_id)
);
CREATE TABLE IF NOT EXISTS trip_sessions (
  id BIGSERIAL PRIMARY KEY,
  trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  started_at TIMESTAMP NOT NULL,
  ended_at TIMESTAMP,
  status VARCHAR(16) NOT NULL DEFAULT 'live'
);
CREATE TABLE IF NOT EXISTS position_reports (
  id BIGSERIAL PRIMARY KEY,
  session_id BIGINT NOT NULL REFERENCES trip_sessions(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id),
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  speed DOUBLE PRECISION NOT NULL DEFAULT 0,
  accuracy DOUBLE PRECISION NOT NULL DEFAULT 0,
  heading DOUBLE PRECISION NOT NULL DEFAULT 0,
  reported_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pos_session_user_time ON position_reports (session_id, user_id, reported_at DESC);
CREATE TABLE IF NOT EXISTS photos (
  id BIGSERIAL PRIMARY KEY,
  trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  session_id BIGINT REFERENCES trip_sessions(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  file_path TEXT NOT NULL,
  thumb_path TEXT NOT NULL,
  caption TEXT NOT NULL DEFAULT '',
  taken_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS alerts (
  id BIGSERIAL PRIMARY KEY,
  session_id BIGINT NOT NULL REFERENCES trip_sessions(id) ON DELETE CASCADE,
  type VARCHAR(16) NOT NULL,
  payload JSONB NOT NULL,
  created_by BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS alert_acks (
  alert_id BIGINT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id),
  acked_at TIMESTAMP NOT NULL,
  PRIMARY KEY (alert_id, user_id)
);
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(88421001)"); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock(88421001)") }()
	if _, err := conn.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.L.Info("schema migrated")
	return nil
}
