package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	instance *RedisClient
	once     sync.Once
)

type RedisClient struct {
	client redis.UniversalClient
}

func Initialize(addr []string, username, password string, dbStr string) error {
	var initErr error

	once.Do(func() {
		ctx := context.Background()

		db, err := strconv.Atoi(dbStr)
		if err != nil {
			initErr = fmt.Errorf("failed to convert Redis DB string to integer: %w", err)
			return
		}

		rdb := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:    addr,
			Username: username,
			Password: password,
			DB:       db,
		})

		if _, err := rdb.Ping(ctx).Result(); err != nil {
			initErr = fmt.Errorf("redis connection failed: %w", err)
			return
		}

		instance = &RedisClient{client: rdb}
	})

	return initErr
}

func GetRedisClient() *RedisClient {
	if instance == nil {
		panic("redis: Initialize() must be called before using redis package")
	}
	return instance
}
