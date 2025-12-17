package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client redis.UniversalClient
}

func NewRedisClient(addr []string, username, password string, dbStr string) (*RedisClient, error) {
	ctx := context.Background()

	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Redis DB string to integer: %w", err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisClient{
		client: rdb,
	}, nil
}
