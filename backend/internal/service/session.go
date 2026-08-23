package service

import (
	"context"
	"net/http"
	"sync"
	"time"

	"gotravel/internal/apperr"
	"gotravel/internal/model"
	"gotravel/internal/repository"
)

type Session struct {
	Repo *repository.Repos
	Trip *Trip
	mu   sync.Mutex
	last map[string]time.Time
}

func NewSession(repo *repository.Repos, trip *Trip) *Session {
	return &Session{Repo: repo, Trip: trip, last: make(map[string]time.Time)}
}

func (s *Session) Start(ctx context.Context, userID, tripID int64) (*model.Session, error) {
	t, err := s.Trip.mustTripLeader(ctx, userID, tripID)
	if err != nil {
		return nil, err
	}
	if live, err := s.Repo.LiveSessionByTrip(ctx, t.ID); err != nil {
		return nil, err
	} else if live != nil {
		return live, nil
	}
	sess := &model.Session{TripID: t.ID}
	if err := s.Repo.CreateSession(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Session) Get(ctx context.Context, userID, sessionID int64) (*model.Session, *model.Trip, error) {
	sess, err := s.Repo.SessionByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if sess == nil {
		return nil, nil, apperr.New(http.StatusNotFound, apperr.NotFound, "session not found")
	}
	t, err := s.Trip.mustTripMember(ctx, userID, sess.TripID)
	return sess, t, err
}

func (s *Session) End(ctx context.Context, userID, sessionID int64) error {
	sess, trip, err := s.Get(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if err := s.Trip.Team.RequireLeader(ctx, trip.TeamID, userID); err != nil {
		return err
	}
	return s.Repo.EndSession(ctx, sess.ID)
}

func (s *Session) RecordPos(ctx context.Context, sessionID, userID int64, lat, lng, speed, acc, heading float64, at time.Time) {
	key := time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(), at.Second()/5*5, 0, at.Location()).Format(time.RFC3339) + ":" + itoa(sessionID) + ":" + itoa(userID)
	s.mu.Lock()
	if prev, ok := s.last[key[:len(key)/2+8]]; ok && at.Sub(prev) < 5*time.Second {
		s.mu.Unlock()
		return
	}
	s.last[itoa(sessionID)+":"+itoa(userID)] = at
	s.mu.Unlock()
	_ = s.Repo.InsertPosition(ctx, model.Position{
		SessionID: sessionID, UserID: userID, Lat: lat, Lng: lng,
		Speed: speed, Accuracy: acc, Heading: heading, ReportedAt: at,
	})
}

func (s *Session) Positions(ctx context.Context, userID, sessionID int64) ([]model.Position, error) {
	if _, _, err := s.Get(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	return s.Repo.RecentPositions(ctx, sessionID, 400)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
