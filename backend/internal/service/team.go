package service

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"strings"

	"gotravel/internal/apperr"
	"gotravel/internal/model"
	"gotravel/internal/repository"
)

type Team struct {
	Repo *repository.Repos
}

func (s *Team) Create(ctx context.Context, userID int64, name string) (*model.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "team name required (≤64)")
	}
	code, err := randomCode(6)
	if err != nil {
		return nil, err
	}
	t := &model.Team{Name: name, LeaderID: userID, InviteCode: code, Role: "leader"}
	if err := s.Repo.CreateTeam(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Team) Join(ctx context.Context, userID int64, code string) (*model.Team, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "invite code must be 6 chars")
	}
	t, err := s.Repo.TeamByInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.New(http.StatusNotFound, apperr.NotFound, "invite not found")
	}
	if err := s.Repo.JoinTeam(ctx, t.ID, userID); err != nil {
		return nil, err
	}
	t.Role = "member"
	if t.LeaderID == userID {
		t.Role = "leader"
	}
	return t, nil
}

func (s *Team) List(ctx context.Context, userID int64) ([]model.Team, error) {
	return s.Repo.TeamsOfUser(ctx, userID)
}

func (s *Team) Get(ctx context.Context, userID, teamID int64) (*model.Team, []model.Member, error) {
	t, err := s.Repo.TeamByID(ctx, teamID)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, apperr.New(http.StatusNotFound, apperr.NotFound, "team not found")
	}
	role, err := s.Repo.MemberRole(ctx, teamID, userID)
	if err != nil {
		return nil, nil, err
	}
	if role == "" {
		return nil, nil, apperr.New(http.StatusForbidden, apperr.Forbidden, "not a member")
	}
	t.Role = role
	ms, err := s.Repo.Members(ctx, teamID)
	return t, ms, err
}

func (s *Team) RequireMember(ctx context.Context, teamID, userID int64) (string, error) {
	role, err := s.Repo.MemberRole(ctx, teamID, userID)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", apperr.New(http.StatusForbidden, apperr.Forbidden, "not a member")
	}
	return role, nil
}

func (s *Team) RequireLeader(ctx context.Context, teamID, userID int64) error {
	role, err := s.RequireMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if role != "leader" {
		return apperr.New(http.StatusForbidden, apperr.Forbidden, "leader only")
	}
	return nil
}

func randomCode(n int) (string, error) {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	for i := range b {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[v.Int64()]
	}
	return string(b), nil
}
