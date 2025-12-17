package redis

import (
	"context"
	"fmt"
)

func InvalidateItem(ctx context.Context, domain string, id string) error {
	rdb := GetRedisClient()
	key := BuildKeyItem(domain, id)
	return rdb.client.Del(ctx, key).Err()
}

func InvalidateByDependency(ctx context.Context, domain string, id string) error {
	rdb := GetRedisClient()
	depKey := BuildKeyDep(domain, id)

	keys, err := rdb.client.SMembers(ctx, depKey).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	pipe := rdb.client.Pipeline()
	pipe.Del(ctx, keys...)
	pipe.Del(ctx, depKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("dependency invalidation failed: %w", err)
	}

	return nil
}

func InvalidateItemWithDep(ctx context.Context, domain string, id string) error {
	rdb := GetRedisClient()

	keysToDelete := []string{}
	itemKey := BuildKeyItem(domain, id)
	keysToDelete = append(keysToDelete, itemKey)

	depKey := BuildKeyDep(domain, id)

	dependents, _ := rdb.client.SMembers(ctx, depKey).Result()
	if len(dependents) > 0 {
		keysToDelete = append(keysToDelete, dependents...)
	}

	pipe := rdb.client.Pipeline()
	pipe.Del(ctx, keysToDelete...)
	pipe.Del(ctx, depKey)

	_, err := pipe.Exec(ctx)
	return err
}

func InvalidateAllCache(ctx context.Context) error {
	rdb := GetRedisClient()
	return rdb.client.FlushDB(ctx).Err()
}

func InvalidateAllViews(ctx context.Context, domain string, identifier string) error {
	rdb := GetRedisClient()
	pattern := BuildKeyItemWildCard(domain, identifier)

	// SCAN ile bul ve sil
	iter := rdb.client.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return rdb.client.Unlink(ctx, keys...).Err()
	}

	return nil
}
