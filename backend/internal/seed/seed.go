package seed

import (
	"context"

	"golang.org/x/crypto/bcrypt"
	"gotravel/internal/logger"
	"gotravel/internal/model"
	"gotravel/internal/repository"
	"gotravel/internal/routegraph"
)

type Account struct {
	Username string
	Password string
	Nickname string
	Color    string
}

func DefaultAccounts() []Account {
	return []Account{
		{"captain", "captain123", "队长阿凯", "#2F6F4E"},
		{"member1", "member123", "小鹿", "#C46A2B"},
		{"member2", "member123", "老周", "#1F4E79"},
	}
}

func Run(ctx context.Context, repo *repository.Repos) error {
	if u, err := repo.UserByUsername(ctx, "captain"); err != nil {
		return err
	} else if u != nil {
		return nil
	}
	ids := map[string]int64{}
	for _, a := range DefaultAccounts() {
		hash, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u := &model.User{Username: a.Username, Password: string(hash), Nickname: a.Nickname, AvatarColor: a.Color}
		if err := repo.CreateUser(ctx, u); err != nil {
			return err
		}
		ids[a.Username] = u.ID
	}
	t := &model.Team{Name: "西湖夜徒小队", LeaderID: ids["captain"], InviteCode: "HTK8M2"}
	if err := repo.CreateTeam(ctx, t); err != nil {
		return err
	}
	_ = repo.JoinTeam(ctx, t.ID, ids["member1"])
	_ = repo.JoinTeam(ctx, t.ID, ids["member2"])
	trip := &model.Trip{TeamID: t.ID, Title: "西湖西线夜徒", Description: "断桥 → 孤山 → 西泠 → 岳庙 → 曲院风荷，含住宿点"}
	if err := repo.CreateTrip(ctx, trip); err != nil {
		return err
	}
	wps := []model.Waypoint{
		{TripID: trip.ID, Seq: 1, Name: "断桥残雪", Kind: model.KindCheckin, Lat: 30.2578, Lng: 120.1512, Note: "集合点"},
		{TripID: trip.ID, Seq: 2, Name: "白堤中段", Kind: model.KindStop, Lat: 30.2549, Lng: 120.1480, PlannedStayMin: 10},
		{TripID: trip.ID, Seq: 3, Name: "孤山公园", Kind: model.KindCheckin, Lat: 30.2532, Lng: 120.1398},
		{TripID: trip.ID, Seq: 4, Name: "西泠印社", Kind: model.KindCheckin, Lat: 30.2515, Lng: 120.1366},
		{TripID: trip.ID, Seq: 5, Name: "岳王庙", Kind: model.KindStop, Lat: 30.2528, Lng: 120.1309, PlannedStayMin: 20},
		{TripID: trip.ID, Seq: 6, Name: "曲院风荷", Kind: model.KindCheckin, Lat: 30.2501, Lng: 120.1284},
		{TripID: trip.ID, Seq: 7, Name: "杭州西溪民宿", Kind: model.KindLodging, Lat: 30.2720, Lng: 120.0635, Note: "当晚住宿"},
	}
	for i := range wps {
		if err := repo.InsertWaypoint(ctx, &wps[i]); err != nil {
			return err
		}
	}
	_ = repo.AddDep(ctx, model.Dep{TripID: trip.ID, FromID: wps[0].ID, ToID: wps[2].ID})
	_ = repo.AddDep(ctx, model.Dep{TripID: trip.ID, FromID: wps[2].ID, ToID: wps[5].ID})
	pts := make([]routegraph.Point, len(wps))
	for i, w := range wps {
		pts[i] = routegraph.Point{Lat: w.Lat, Lng: w.Lng}
	}
	dist, _ := (routegraph.HaversineProvider{}).Compute(pts)
	trip.TotalDistanceM = dist.TotalMeters
	_ = repo.UpdateTrip(ctx, trip)
	logger.L.Info("seed ready", "team", t.ID, "trip", trip.ID, "invite", t.InviteCode)
	return nil
}
