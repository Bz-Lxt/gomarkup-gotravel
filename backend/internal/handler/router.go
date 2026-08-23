package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"gotravel/internal/apperr"
	"gotravel/internal/config"
	"gotravel/internal/middleware"
	"gotravel/internal/model"
	"gotravel/internal/response"
	"gotravel/internal/service"
	"gotravel/internal/simulator"
	"gotravel/internal/ws"
)

type API struct {
	Cfg     config.Config
	Auth    *service.Auth
	Team    *service.Team
	Trip    *service.Trip
	Sess    *service.Session
	Photo   *service.Photo
	Alert   *service.Alert
	Hub     *ws.Hub
	Sim     *simulator.GPS
	GeoName string
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.health)
	mux.HandleFunc("GET /api/v1/metrics", a.metrics)
	mux.HandleFunc("POST /api/v1/auth/register", a.register)
	mux.HandleFunc("POST /api/v1/auth/login", a.login)

	prot := middleware.Auth(a.Auth)
	mux.Handle("GET /api/v1/auth/me", prot(http.HandlerFunc(a.me)))
	mux.Handle("POST /api/v1/teams", prot(http.HandlerFunc(a.createTeam)))
	mux.Handle("GET /api/v1/teams", prot(http.HandlerFunc(a.listTeams)))
	mux.Handle("POST /api/v1/teams/join", prot(http.HandlerFunc(a.joinTeam)))
	mux.Handle("GET /api/v1/teams/{id}", prot(http.HandlerFunc(a.getTeam)))
	mux.Handle("POST /api/v1/teams/{id}/trips", prot(http.HandlerFunc(a.createTrip)))
	mux.Handle("GET /api/v1/teams/{id}/trips", prot(http.HandlerFunc(a.listTrips)))
	mux.Handle("GET /api/v1/trips/{id}", prot(http.HandlerFunc(a.getTrip)))
	mux.Handle("PATCH /api/v1/trips/{id}", prot(http.HandlerFunc(a.patchTrip)))
	mux.Handle("DELETE /api/v1/trips/{id}", prot(http.HandlerFunc(a.deleteTrip)))
	mux.Handle("POST /api/v1/trips/{id}/waypoints", prot(http.HandlerFunc(a.addWP)))
	mux.Handle("PUT /api/v1/trips/{id}/waypoints/reorder", prot(http.HandlerFunc(a.reorder)))
	mux.Handle("POST /api/v1/trips/{id}/waypoints/import", prot(http.HandlerFunc(a.importGJ)))
	mux.Handle("POST /api/v1/trips/{id}/deps", prot(http.HandlerFunc(a.addDep)))
	mux.Handle("DELETE /api/v1/trips/{id}/deps", prot(http.HandlerFunc(a.delDep)))
	mux.Handle("GET /api/v1/trips/{id}/topo", prot(http.HandlerFunc(a.topo)))
	mux.Handle("GET /api/v1/trips/{id}/distance", prot(http.HandlerFunc(a.distance)))
	mux.Handle("PATCH /api/v1/waypoints/{id}", prot(http.HandlerFunc(a.patchWP)))
	mux.Handle("DELETE /api/v1/waypoints/{id}", prot(http.HandlerFunc(a.delWP)))
	mux.Handle("POST /api/v1/trips/{id}/sessions", prot(http.HandlerFunc(a.startSess)))
	mux.Handle("GET /api/v1/sessions/{id}", prot(http.HandlerFunc(a.getSess)))
	mux.Handle("POST /api/v1/sessions/{id}/end", prot(http.HandlerFunc(a.endSess)))
	mux.Handle("GET /api/v1/sessions/{id}/positions", prot(http.HandlerFunc(a.positions)))
	mux.Handle("POST /api/v1/sessions/{id}/rally", prot(http.HandlerFunc(a.rally)))
	mux.Handle("GET /api/v1/sessions/{id}/alerts", prot(http.HandlerFunc(a.alerts)))
	mux.Handle("POST /api/v1/alerts/{id}/ack", prot(http.HandlerFunc(a.ack)))
	mux.Handle("POST /api/v1/trips/{id}/photos", prot(http.HandlerFunc(a.uploadPhoto)))
	mux.Handle("GET /api/v1/trips/{id}/photos", prot(http.HandlerFunc(a.listPhotos)))
	mux.Handle("POST /api/v1/sim/start", prot(http.HandlerFunc(a.simStart)))
	mux.Handle("POST /api/v1/sim/stop", prot(http.HandlerFunc(a.simStop)))
	mux.Handle("GET /api/v1/sim/status", prot(http.HandlerFunc(a.simStatus)))
	mux.Handle("GET /ws", prot(http.HandlerFunc(a.ws)))
	return mux
}

func uid(r *http.Request) int64 {
	u, _ := middleware.UserFrom(r.Context())
	return u.ID
}

func pathID(r *http.Request, name string) (int64, error) {
	v, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || v <= 0 {
		return 0, apperr.New(http.StatusBadRequest, apperr.Validation, name+" invalid")
	}
	return v, nil
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, 200, map[string]any{
		"service": "gotravel", "tz": "Asia/Shanghai",
		"geo": a.GeoName, "gps": a.Cfg.GPSProvider, "route": a.Cfg.RouteProvider,
	})
}

func (a *API) metrics(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, 200, a.Hub.Snapshot())
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	u, tok, err := a.Auth.Register(r.Context(), req.Username, req.Password, req.Nickname)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, map[string]any{"user": publicUser(u), "token": tok})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	u, tok, err := a.Auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"user": publicUser(u), "token": tok})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.Auth.Repo.UserByID(r.Context(), uid(r))
	if err != nil || u == nil {
		response.Fail(w, apperr.New(401, apperr.Unauthorized, "user gone"))
		return
	}
	response.JSON(w, 200, publicUser(u))
}

func publicUser(u *model.User) map[string]any {
	return map[string]any{"id": u.ID, "username": u.Username, "nickname": u.Nickname, "avatar_color": u.AvatarColor}
}

func (a *API) createTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	t, err := a.Team.Create(r.Context(), uid(r), req.Name)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, t)
}

func (a *API) listTeams(w http.ResponseWriter, r *http.Request) {
	ts, err := a.Team.List(r.Context(), uid(r))
	if err != nil {
		response.Fail(w, err)
		return
	}
	if ts == nil {
		ts = []model.Team{}
	}
	response.JSON(w, 200, ts)
}

func (a *API) joinTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	t, err := a.Team.Join(r.Context(), uid(r), req.Code)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, t)
}

func (a *API) getTeam(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	t, ms, err := a.Team.Get(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"team": t, "members": ms})
}

func (a *API) createTrip(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	t, err := a.Trip.Create(r.Context(), uid(r), teamID, req.Title, req.Description)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, t)
}

func (a *API) listTrips(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	ts, err := a.Trip.List(r.Context(), uid(r), teamID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if ts == nil {
		ts = []model.Trip{}
	}
	response.JSON(w, 200, ts)
}

func (a *API) getTrip(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	t, wps, deps, err := a.Trip.Get(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"trip": t, "waypoints": wps, "deps": deps})
}

func (a *API) patchTrip(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var t model.Trip
	if err := response.Decode(r, &t); err != nil {
		response.Fail(w, err)
		return
	}
	t.ID = id
	out, err := a.Trip.Update(r.Context(), uid(r), &t)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, out)
}

func (a *API) deleteTrip(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := a.Trip.Delete(r.Context(), uid(r), id); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]bool{"deleted": true})
}

func (a *API) addWP(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var wp model.Waypoint
	if err := response.Decode(r, &wp); err != nil {
		response.Fail(w, err)
		return
	}
	wp.TripID = id
	out, dist, err := a.Trip.AddWaypoint(r.Context(), uid(r), &wp)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, map[string]any{"waypoint": out, "distance": dist})
}

func (a *API) reorder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	dist, err := a.Trip.Reorder(r.Context(), uid(r), id, req.IDs)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, dist)
}

func (a *API) importGJ(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.Fail(w, apperr.New(400, apperr.BadRequest, "read body"))
		return
	}
	wps, dist, err := a.Trip.ImportGeoJSON(r.Context(), uid(r), id, raw)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, map[string]any{"waypoints": wps, "distance": dist})
}

func (a *API) addDep(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req struct {
		From int64 `json:"from_waypoint_id"`
		To   int64 `json:"to_waypoint_id"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	order, err := a.Trip.AddDep(r.Context(), uid(r), id, req.From, req.To)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, map[string]any{"topo": order})
}

func (a *API) delDep(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	to, _ := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
	if err := a.Trip.RemoveDep(r.Context(), uid(r), id, from, to); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]bool{"deleted": true})
}

func (a *API) topo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	order, deps, err := a.Trip.Topo(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"topo": order, "deps": deps})
}

func (a *API) distance(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	d, err := a.Trip.Distance(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, d)
}

func (a *API) patchWP(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var wp model.Waypoint
	if err := response.Decode(r, &wp); err != nil {
		response.Fail(w, err)
		return
	}
	wp.ID = id
	out, dist, err := a.Trip.PatchWaypoint(r.Context(), uid(r), &wp)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"waypoint": out, "distance": dist})
}

func (a *API) delWP(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	dist, err := a.Trip.DeleteWaypoint(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, dist)
}

func (a *API) startSess(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	s, err := a.Sess.Start(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, s)
}

func (a *API) getSess(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	s, t, err := a.Sess.Get(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]any{"session": s, "trip": t})
}

func (a *API) endSess(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := a.Sess.End(r.Context(), uid(r), id); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]bool{"ended": true})
}

func (a *API) positions(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	ps, err := a.Sess.Positions(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, ps)
}

func (a *API) rally(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Message string  `json:"message"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	al, err := a.Alert.Rally(r.Context(), uid(r), id, req.Lat, req.Lng, req.Message)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, al)
}

func (a *API) alerts(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	list, err := a.Alert.List(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, al := range list {
		var payload any
		_ = json.Unmarshal(al.Payload, &payload)
		out = append(out, map[string]any{
			"id": al.ID, "session_id": al.SessionID, "type": al.Type,
			"payload": payload, "created_by": al.CreatedBy, "acks": al.Acks,
		})
	}
	response.JSON(w, 200, out)
}

func (a *API) ack(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := a.Alert.Ack(r.Context(), uid(r), id); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, map[string]bool{"acked": true})
}

func (a *API) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20+4096)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.Fail(w, apperr.New(400, apperr.PayloadTooBig, "multipart too large"))
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		response.Fail(w, apperr.New(400, apperr.Validation, "file required"))
		return
	}
	defer file.Close()
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("lng"), 64)
	var sid *int64
	if v := strings.TrimSpace(r.FormValue("session_id")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			sid = &n
		}
	}
	p, err := a.Photo.Save(r.Context(), uid(r), id, sid, lat, lng, r.FormValue("caption"), hdr.Filename, file, hdr.Size)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 201, p)
}

func (a *API) listPhotos(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Fail(w, err)
		return
	}
	ps, err := a.Photo.List(r.Context(), uid(r), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, 200, ps)
}

func (a *API) simStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID int64 `json:"session_id"`
		Count     int   `json:"count"`
		Laggard   bool  `json:"laggard"`
	}
	if err := response.Decode(r, &req); err != nil {
		response.Fail(w, err)
		return
	}
	if _, _, err := a.Sess.Get(r.Context(), uid(r), req.SessionID); err != nil {
		response.Fail(w, err)
		return
	}
	tripID := int64(0)
	if sess, err := a.Sess.Repo.SessionByID(r.Context(), req.SessionID); err == nil && sess != nil {
		tripID = sess.TripID
	}
	wps, _ := a.Sess.Repo.Waypoints(r.Context(), tripID)
	pts := make([][2]float64, 0, len(wps))
	for _, wp := range wps {
		pts = append(pts, [2]float64{wp.Lat, wp.Lng})
	}
	if err := a.Sim.Start(req.SessionID, pts, req.Count, req.Laggard); err != nil {
		response.Fail(w, apperr.New(400, apperr.Validation, err.Error()))
		return
	}
	response.JSON(w, 200, a.Sim.Status())
}

func (a *API) simStop(w http.ResponseWriter, r *http.Request) {
	a.Sim.Stop()
	response.JSON(w, 200, a.Sim.Status())
}

func (a *API) simStatus(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, 200, a.Sim.Status())
}

func (a *API) ws(w http.ResponseWriter, r *http.Request) {
	sid, err := strconv.ParseInt(r.URL.Query().Get("session_id"), 10, 64)
	if err != nil || sid <= 0 {
		response.Fail(w, apperr.New(400, apperr.Validation, "session_id required"))
		return
	}
	sess, _, err := a.Sess.Get(r.Context(), uid(r), sid)
	if err != nil {
		response.Fail(w, err)
		return
	}
	u, err := a.Auth.Repo.UserByID(r.Context(), uid(r))
	if err != nil || u == nil {
		response.Fail(w, apperr.New(401, apperr.Unauthorized, "user gone"))
		return
	}
	role := "member"
	if trip, e := a.Trip.Repo.TripByID(r.Context(), sess.TripID); e == nil && trip != nil {
		if rl, e2 := a.Team.Repo.MemberRole(r.Context(), trip.TeamID, u.ID); e2 == nil && rl != "" {
			role = rl
		}
	}
	a.Hub.ServeWS(w, r, sid, u.ID, u.Nickname, u.AvatarColor, role)
}
