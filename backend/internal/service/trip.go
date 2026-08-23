package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"gotravel/internal/apperr"
	"gotravel/internal/geo"
	"gotravel/internal/model"
	"gotravel/internal/repository"
	"gotravel/internal/routegraph"
)

type Trip struct {
	Repo     *repository.Repos
	Team     *Team
	DistProv routegraph.Provider
}

func (s *Trip) Create(ctx context.Context, userID, teamID int64, title, desc string) (*model.Trip, error) {
	if err := s.Team.RequireLeader(ctx, teamID, userID); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 80 {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "title required (≤80)")
	}
	t := &model.Trip{TeamID: teamID, Title: title, Description: strings.TrimSpace(desc)}
	if err := s.Repo.CreateTrip(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Trip) List(ctx context.Context, userID, teamID int64) ([]model.Trip, error) {
	if _, err := s.Team.RequireMember(ctx, teamID, userID); err != nil {
		return nil, err
	}
	return s.Repo.TripsByTeam(ctx, teamID)
}

func (s *Trip) Get(ctx context.Context, userID, tripID int64) (*model.Trip, []model.Waypoint, []model.Dep, error) {
	t, err := s.mustTripMember(ctx, userID, tripID)
	if err != nil {
		return nil, nil, nil, err
	}
	wps, err := s.Repo.Waypoints(ctx, tripID)
	if err != nil {
		return nil, nil, nil, err
	}
	deps, err := s.Repo.Deps(ctx, tripID)
	return t, wps, deps, err
}

func (s *Trip) mustTripMember(ctx context.Context, userID, tripID int64) (*model.Trip, error) {
	t, err := s.Repo.TripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.New(http.StatusNotFound, apperr.NotFound, "trip not found")
	}
	if _, err := s.Team.RequireMember(ctx, t.TeamID, userID); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Trip) mustTripLeader(ctx context.Context, userID, tripID int64) (*model.Trip, error) {
	t, err := s.Repo.TripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.New(http.StatusNotFound, apperr.NotFound, "trip not found")
	}
	if err := s.Team.RequireLeader(ctx, t.TeamID, userID); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Trip) Update(ctx context.Context, userID int64, t *model.Trip) (*model.Trip, error) {
	cur, err := s.mustTripLeader(ctx, userID, t.ID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(t.Title) != "" {
		cur.Title = strings.TrimSpace(t.Title)
	}
	cur.Description = t.Description
	if t.Status != "" {
		cur.Status = t.Status
	}
	if err := s.Repo.UpdateTrip(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Trip) Delete(ctx context.Context, userID, tripID int64) error {
	if _, err := s.mustTripLeader(ctx, userID, tripID); err != nil {
		return err
	}
	return s.Repo.DeleteTrip(ctx, tripID)
}

func validKind(k string) bool {
	return k == model.KindStop || k == model.KindCheckin || k == model.KindLodging
}

func (s *Trip) AddWaypoint(ctx context.Context, userID int64, w *model.Waypoint) (*model.Waypoint, *routegraph.DistanceResult, error) {
	if _, err := s.mustTripLeader(ctx, userID, w.TripID); err != nil {
		return nil, nil, err
	}
	if !geo.ValidCoord(w.Lat, w.Lng) {
		return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "lat/lng out of range")
	}
	w.Lat, w.Lng = geo.ClampCoord(w.Lat, w.Lng)
	w.Kind = strings.ToUpper(strings.TrimSpace(w.Kind))
	if !validKind(w.Kind) {
		return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "kind must be STOP|CHECKIN|LODGING")
	}
	if strings.TrimSpace(w.Name) == "" {
		w.Name = "未命名点"
	}
	seq, err := s.Repo.NextSeq(ctx, w.TripID)
	if err != nil {
		return nil, nil, err
	}
	w.Seq = seq
	if err := s.Repo.InsertWaypoint(ctx, w); err != nil {
		return nil, nil, err
	}
	dist, err := s.recalc(ctx, w.TripID)
	return w, dist, err
}

func (s *Trip) PatchWaypoint(ctx context.Context, userID int64, w *model.Waypoint) (*model.Waypoint, *routegraph.DistanceResult, error) {
	cur, err := s.Repo.WaypointByID(ctx, w.ID)
	if err != nil {
		return nil, nil, err
	}
	if cur == nil {
		return nil, nil, apperr.New(http.StatusNotFound, apperr.NotFound, "waypoint not found")
	}
	if _, err := s.mustTripLeader(ctx, userID, cur.TripID); err != nil {
		return nil, nil, err
	}
	if w.Name != "" {
		cur.Name = w.Name
	}
	if w.Kind != "" {
		cur.Kind = strings.ToUpper(w.Kind)
		if !validKind(cur.Kind) {
			return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "invalid kind")
		}
	}
	if w.Lat != 0 || w.Lng != 0 {
		if !geo.ValidCoord(w.Lat, w.Lng) {
			return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "lat/lng out of range")
		}
		cur.Lat, cur.Lng = geo.ClampCoord(w.Lat, w.Lng)
	}
	cur.Note = w.Note
	cur.PlannedStayMin = w.PlannedStayMin
	if err := s.Repo.UpdateWaypoint(ctx, cur); err != nil {
		return nil, nil, err
	}
	dist, err := s.recalc(ctx, cur.TripID)
	return cur, dist, err
}

func (s *Trip) DeleteWaypoint(ctx context.Context, userID, wpID int64) (*routegraph.DistanceResult, error) {
	cur, err := s.Repo.WaypointByID(ctx, wpID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, apperr.New(http.StatusNotFound, apperr.NotFound, "waypoint not found")
	}
	if _, err := s.mustTripLeader(ctx, userID, cur.TripID); err != nil {
		return nil, err
	}
	if err := s.Repo.DeleteWaypoint(ctx, wpID); err != nil {
		return nil, err
	}
	return s.recalc(ctx, cur.TripID)
}

func (s *Trip) Reorder(ctx context.Context, userID, tripID int64, ids []int64) (*routegraph.DistanceResult, error) {
	if _, err := s.mustTripLeader(ctx, userID, tripID); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "ids required")
	}
	if err := s.Repo.ReorderWaypoints(ctx, tripID, ids); err != nil {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, err.Error())
	}
	return s.recalc(ctx, tripID)
}

func (s *Trip) ImportGeoJSON(ctx context.Context, userID, tripID int64, raw []byte) ([]model.Waypoint, *routegraph.DistanceResult, error) {
	if _, err := s.mustTripLeader(ctx, userID, tripID); err != nil {
		return nil, nil, err
	}
	var fc struct {
		Type     string `json:"type"`
		Features []struct {
			Type       string `json:"type"`
			Properties struct {
				Name string `json:"name"`
				Kind string `json:"kind"`
			} `json:"properties"`
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &fc); err != nil || fc.Type != "FeatureCollection" {
		return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "must be GeoJSON FeatureCollection")
	}
	if len(fc.Features) == 0 {
		return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "features empty")
	}
	var created []model.Waypoint
	for _, f := range fc.Features {
		if f.Geometry.Type != "Point" {
			continue
		}
		var coord []float64
		if err := json.Unmarshal(f.Geometry.Coordinates, &coord); err != nil || len(coord) < 2 {
			return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "point coordinates must be [lng,lat]")
		}
		lng, lat := coord[0], coord[1]
		if !geo.ValidCoord(lat, lng) {
			return nil, nil, apperr.New(http.StatusBadRequest, apperr.Validation, "invalid coordinate in geojson")
		}
		kind := strings.ToUpper(f.Properties.Kind)
		if !validKind(kind) {
			kind = model.KindStop
		}
		w := &model.Waypoint{TripID: tripID, Name: f.Properties.Name, Kind: kind, Lat: lat, Lng: lng}
		if w.Name == "" {
			w.Name = "导入点"
		}
		seq, err := s.Repo.NextSeq(ctx, tripID)
		if err != nil {
			return nil, nil, err
		}
		w.Seq = seq
		if err := s.Repo.InsertWaypoint(ctx, w); err != nil {
			return nil, nil, err
		}
		created = append(created, *w)
	}
	dist, err := s.recalc(ctx, tripID)
	return created, dist, err
}

func (s *Trip) AddDep(ctx context.Context, userID, tripID, from, to int64) ([]int64, error) {
	if _, err := s.mustTripLeader(ctx, userID, tripID); err != nil {
		return nil, err
	}
	wps, err := s.Repo.Waypoints(ctx, tripID)
	if err != nil {
		return nil, err
	}
	g := routegraph.New()
	seen := map[int64]bool{}
	for _, w := range wps {
		g.AddNode(routegraph.NodeID(w.ID))
		seen[w.ID] = true
	}
	if !seen[from] || !seen[to] {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "waypoints not in trip")
	}
	deps, err := s.Repo.Deps(ctx, tripID)
	if err != nil {
		return nil, err
	}
	for _, d := range deps {
		if err := g.AddEdge(routegraph.NodeID(d.FromID), routegraph.NodeID(d.ToID)); err != nil {
			return nil, err
		}
	}
	if err := g.AddEdge(routegraph.NodeID(from), routegraph.NodeID(to)); err != nil {
		if ce, ok := err.(routegraph.CycleError); ok {
			ids := make([]int64, len(ce.Path))
			for i, n := range ce.Path {
				ids[i] = int64(n)
			}
			return nil, apperr.WithDetail(http.StatusConflict, apperr.CycleDetected, "dependency cycle", ids)
		}
		return nil, apperr.New(http.StatusConflict, apperr.Conflict, err.Error())
	}
	if err := s.Repo.AddDep(ctx, model.Dep{TripID: tripID, FromID: from, ToID: to}); err != nil {
		return nil, err
	}
	order, err := g.TopoSort()
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(order))
	for i, n := range order {
		out[i] = int64(n)
	}
	return out, nil
}

func (s *Trip) RemoveDep(ctx context.Context, userID, tripID, from, to int64) error {
	if _, err := s.mustTripLeader(ctx, userID, tripID); err != nil {
		return err
	}
	return s.Repo.DeleteDep(ctx, tripID, from, to)
}

func (s *Trip) Topo(ctx context.Context, userID, tripID int64) ([]int64, []model.Dep, error) {
	if _, err := s.mustTripMember(ctx, userID, tripID); err != nil {
		return nil, nil, err
	}
	wps, err := s.Repo.Waypoints(ctx, tripID)
	if err != nil {
		return nil, nil, err
	}
	deps, err := s.Repo.Deps(ctx, tripID)
	if err != nil {
		return nil, nil, err
	}
	g := routegraph.New()
	for _, w := range wps {
		g.AddNode(routegraph.NodeID(w.ID))
	}
	for _, d := range deps {
		if err := g.AddEdge(routegraph.NodeID(d.FromID), routegraph.NodeID(d.ToID)); err != nil {
			if ce, ok := err.(routegraph.CycleError); ok {
				ids := make([]int64, len(ce.Path))
				for i, n := range ce.Path {
					ids[i] = int64(n)
				}
				return nil, deps, apperr.WithDetail(http.StatusConflict, apperr.CycleDetected, "cycle", ids)
			}
			return nil, deps, err
		}
	}
	order, err := g.TopoSort()
	if err != nil {
		return nil, deps, err
	}
	out := make([]int64, len(order))
	for i, n := range order {
		out[i] = int64(n)
	}
	return out, deps, nil
}

func (s *Trip) Distance(ctx context.Context, userID, tripID int64) (*routegraph.DistanceResult, error) {
	if _, err := s.mustTripMember(ctx, userID, tripID); err != nil {
		return nil, err
	}
	return s.recalc(ctx, tripID)
}

func (s *Trip) recalc(ctx context.Context, tripID int64) (*routegraph.DistanceResult, error) {
	wps, err := s.Repo.Waypoints(ctx, tripID)
	if err != nil {
		return nil, err
	}
	pts := make([]routegraph.Point, len(wps))
	for i, w := range wps {
		pts[i] = routegraph.Point{Lat: w.Lat, Lng: w.Lng}
	}
	prov := s.DistProv
	if prov == nil {
		prov = routegraph.HaversineProvider{}
	}
	res, err := prov.Compute(pts)
	if err != nil {
		return nil, err
	}
	t, err := s.Repo.TripByID(ctx, tripID)
	if err != nil || t == nil {
		return &res, err
	}
	t.TotalDistanceM = res.TotalMeters
	_ = s.Repo.UpdateTrip(ctx, t)
	return &res, nil
}
