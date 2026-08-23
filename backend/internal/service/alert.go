package service

import (
	"context"
	"encoding/json"
	"net/http"

	"gotravel/internal/apperr"
	"gotravel/internal/geo"
	"gotravel/internal/model"
	"gotravel/internal/repository"
	"gotravel/internal/timeutil"
	"gotravel/internal/ws"
)

type Alert struct {
	Repo    *repository.Repos
	Session *Session
	Hub     *ws.Hub
}

func (s *Alert) Rally(ctx context.Context, userID, sessionID int64, lat, lng float64, message string) (*model.Alert, error) {
	sess, trip, err := s.Session.Get(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.Session.Trip.Team.RequireLeader(ctx, trip.TeamID, userID); err != nil {
		return nil, err
	}
	if !geo.ValidCoord(lat, lng) {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "rally coordinate invalid")
	}
	if message == "" {
		message = "紧急集合，立刻到队长位置会合"
	}
	payload, _ := json.Marshal(map[string]any{
		"lat": lat, "lng": lng, "message": message,
	})
	a := &model.Alert{SessionID: sess.ID, Type: "RALLY", Payload: payload, CreatedBy: userID}
	if err := s.Repo.InsertAlert(ctx, a); err != nil {
		return nil, err
	}
	if s.Hub != nil {
		s.Hub.BroadcastPrio(sessionID, ws.MustJSON(ws.RallyOut{
			Type: ws.TypeRally, AlertID: a.ID, Lat: lat, Lng: lng,
			Message: message, CreatedAt: timeutil.Format(timeutil.Now()),
		}))
	}
	return a, nil
}

func (s *Alert) PersistLaggard(ctx context.Context, sessionID int64, ga *geo.Alert) (*model.Alert, error) {
	payload, _ := json.Marshal(map[string]any{
		"user_id": ga.UserID, "distance_m": ga.DistanceM, "mode": ga.Mode,
		"lat": ga.Lat, "lng": ga.Lng,
	})
	a := &model.Alert{SessionID: sessionID, Type: "LAGGARD", Payload: payload, CreatedBy: ga.UserID}
	if err := s.Repo.InsertAlert(ctx, a); err != nil {
		return nil, err
	}
	u, _ := s.Repo.UserByID(ctx, ga.UserID)
	nick := ""
	if u != nil {
		nick = u.Nickname
	}
	if s.Hub != nil {
		s.Hub.BroadcastPrio(sessionID, ws.MustJSON(ws.LaggardOut{
			Type: ws.TypeLaggard, AlertID: a.ID, UserID: ga.UserID, Nickname: nick,
			DistanceM: ga.DistanceM, Mode: string(ga.Mode), Lat: ga.Lat, Lng: ga.Lng,
		}))
	}
	return a, nil
}

func (s *Alert) List(ctx context.Context, userID, sessionID int64) ([]model.Alert, error) {
	if _, _, err := s.Session.Get(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	return s.Repo.Alerts(ctx, sessionID)
}

func (s *Alert) Ack(ctx context.Context, userID, alertID int64) error {
	if alertID <= 0 {
		return apperr.New(http.StatusBadRequest, apperr.Validation, "alert_id required")
	}
	return s.Repo.AckAlert(ctx, alertID, userID)
}
