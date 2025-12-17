package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

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
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			initErr = fmt.Errorf("redis db parse error: %w", err)
			return
		}

		rdb := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:        addr,
			Username:     username,
			Password:     password,
			DB:           db,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := rdb.Ping(ctx).Result(); err != nil {
			initErr = fmt.Errorf("redis connection failed: %w", err)
			return
		}

		instance = &RedisClient{client: rdb}
	})

	return initErr
}

func GetClient() *RedisClient {
	if instance == nil {
		panic("redis not initialized")
	}
	return instance
}

// ═══════════════════════════════════════════════════════════════════
// HELPER METHODS (Stats/Debug)
// ═══════════════════════════════════════════════════════════════════

// Scan - Key pattern araması için (stats endpoint'inde kullanılıyor)
func (r *RedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	return r.client.Scan(ctx, cursor, match, count)
}

// SMembers - Set içeriğini almak için (dependency görüntüleme)
func (r *RedisClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return r.client.SMembers(ctx, key)
}

// DBSize - Toplam key sayısı
func (r *RedisClient) DBSize(ctx context.Context) *redis.IntCmd {
	return r.client.DBSize(ctx)
}
