package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gotravel/internal/logger"
)

func Connect(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 3 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.L.Warn("redis unavailable, geo will stay on rtree", "err", err)
		return rdb, err
	}
	logger.L.Info("redis connected")
	return rdb, nil
}
