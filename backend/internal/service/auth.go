package service

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gotravel/internal/apperr"
	"gotravel/internal/model"
	"gotravel/internal/repository"
	"gotravel/internal/timeutil"
)

var avatarPalette = []string{"#2F6F4E", "#C46A2B", "#1F4E79", "#8B3A3A", "#6B4C9A", "#3D7A6A"}

type Auth struct {
	Repo   *repository.Repos
	Secret []byte
}

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"un"`
	jwt.RegisteredClaims
}

func (a *Auth) Register(ctx context.Context, username, password, nickname string) (*model.User, string, error) {
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	if err := validateUsername(username); err != nil {
		return nil, "", err
	}
	if len(password) < 6 || len(password) > 72 {
		return nil, "", apperr.New(http.StatusBadRequest, apperr.Validation, "password must be 6-72 chars")
	}
	if nickname == "" {
		nickname = username
	}
	if exist, err := a.Repo.UserByUsername(ctx, username); err != nil {
		return nil, "", err
	} else if exist != nil {
		return nil, "", apperr.New(http.StatusConflict, apperr.Conflict, "username taken")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	u := &model.User{
		Username:    username,
		Password:    string(hash),
		Nickname:    nickname,
		AvatarColor: avatarPalette[len(username)%len(avatarPalette)],
	}
	if err := a.Repo.CreateUser(ctx, u); err != nil {
		return nil, "", err
	}
	tok, err := a.sign(u)
	return u, tok, err
}

func (a *Auth) Login(ctx context.Context, username, password string) (*model.User, string, error) {
	u, err := a.Repo.UserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, "", err
	}
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return nil, "", apperr.New(http.StatusUnauthorized, apperr.Unauthorized, "invalid credentials")
	}
	tok, err := a.sign(u)
	return u, tok, err
}

func (a *Auth) sign(u *model.User) (string, error) {
	now := timeutil.Now()
	c := Claims{
		UserID:   u.ID,
		Username: u.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(a.Secret)
}

func (a *Auth) Parse(token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return a.Secret, nil
	})
	if err != nil || !t.Valid {
		return nil, apperr.New(http.StatusUnauthorized, apperr.Unauthorized, "invalid token")
	}
	c, ok := t.Claims.(*Claims)
	if !ok {
		return nil, apperr.New(http.StatusUnauthorized, apperr.Unauthorized, "invalid token")
	}
	return c, nil
}

func validateUsername(s string) error {
	if len(s) < 3 || len(s) > 32 {
		return apperr.New(http.StatusBadRequest, apperr.Validation, "username must be 3-32 chars")
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			return apperr.New(http.StatusBadRequest, apperr.Validation, "username allows letters, digits, _-")
		}
	}
	return nil
}
