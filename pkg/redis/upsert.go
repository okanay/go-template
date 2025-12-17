package redis

import (
	"context"
	"encoding/json"
	"time"
)

func UpsertItem[T Cacheable](
	ctx context.Context,
	domain string,
	data T,
	expiration time.Duration,
) error {
	rdb := GetRedisClient()
	id := data.GetID()
	key := BuildKeyItem(domain, id)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	pipe := rdb.client.Pipeline()

	pipe.Set(ctx, key, jsonData, expiration)

	for _, dep := range data.GetDependencies() {
		depKey := BuildKeyDep(dep.Domain, dep.Id)
		pipe.SAdd(ctx, depKey, key)
		pipe.Expire(ctx, depKey, expiration)
	}

	_, err = pipe.Exec(ctx)
	return err
}
