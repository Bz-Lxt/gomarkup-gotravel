package geo

import "context"

type Member struct {
	ID  string
	Lat float64
	Lng float64
}

type Index interface {
	Upsert(ctx context.Context, key string, m Member) error
	Remove(ctx context.Context, key, id string) error
	All(ctx context.Context, key string) ([]Member, error)
	Nearest(ctx context.Context, key string, lat, lng float64, n int) ([]Member, error)
	Farthest(ctx context.Context, key string, lat, lng float64) (Member, float64, error)
	Backend() string
}

type emptyError struct{ msg string }

func (e emptyError) Error() string { return e.msg }

var ErrEmpty = emptyError{"geo index empty"}
