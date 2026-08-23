package geo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisGeoIndex struct {
	rdb *redis.Client
}

func NewRedisGeoIndex(rdb *redis.Client) *RedisGeoIndex {
	return &RedisGeoIndex{rdb: rdb}
}

func (x *RedisGeoIndex) Backend() string { return "redis" }

func keyName(key string) string { return "geo:session:" + key }

func (x *RedisGeoIndex) Upsert(ctx context.Context, key string, m Member) error {
	if !ValidCoord(m.Lat, m.Lng) {
		return emptyError{"invalid coordinate"}
	}
	return x.rdb.GeoAdd(ctx, keyName(key), &redis.GeoLocation{
		Name:      m.ID,
		Longitude: m.Lng,
		Latitude:  m.Lat,
	}).Err()
}

func (x *RedisGeoIndex) Remove(ctx context.Context, key, id string) error {
	return x.rdb.ZRem(ctx, keyName(key), id).Err()
}

func (x *RedisGeoIndex) All(ctx context.Context, key string) ([]Member, error) {
	ids, err := x.rdb.ZRange(ctx, keyName(key), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(ids))
	for _, id := range ids {
		pos, err := x.rdb.GeoPos(ctx, keyName(key), id).Result()
		if err != nil || len(pos) == 0 || pos[0] == nil {
			continue
		}
		out = append(out, Member{ID: id, Lat: pos[0].Latitude, Lng: pos[0].Longitude})
	}
	return out, nil
}

func (x *RedisGeoIndex) Nearest(ctx context.Context, key string, lat, lng float64, n int) ([]Member, error) {
	locs, err := x.rdb.GeoSearchLocation(ctx, keyName(key), &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     20000,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      n,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(locs))
	for _, loc := range locs {
		out = append(out, Member{ID: loc.Name, Lat: loc.Latitude, Lng: loc.Longitude})
	}
	return out, nil
}

func (x *RedisGeoIndex) Farthest(ctx context.Context, key string, lat, lng float64) (Member, float64, error) {
	all, err := x.All(ctx, key)
	if err != nil {
		return Member{}, 0, err
	}
	if len(all) == 0 {
		return Member{}, 0, ErrEmpty
	}
	var best Member
	var bestD float64 = -1
	for _, m := range all {
		d := Haversine(lat, lng, m.Lat, m.Lng)
		if d > bestD {
			bestD = d
			best = m
		}
	}
	return best, bestD, nil
}

func MemberID(userID int64) string { return strconv.FormatInt(userID, 10) }

func ParseMemberID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("member id: %w", err)
	}
	return v, nil
}
