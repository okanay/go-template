package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cacheable interface {
	GetID() string

	GetDependencies() []struct {
		Domain string
		Id     string
	}
}

func GetItem[T Cacheable](
	ctx context.Context,
	domain string,
	id string,
	expiration time.Duration,
	dbFetcher func() (T, error),
) (T, error) {
	rdb := GetRedisClient()
	key := BuildKeyItem(domain, id)

	var result T
	var cacheHit = false

	val, err := rdb.client.Get(ctx, key).Result()

	if err == nil {
		if jsonErr := json.Unmarshal([]byte(val), &result); jsonErr == nil {
			cacheHit = true
		}
	} else if err != redis.Nil {
	}

	if cacheHit {
		return result, nil
	}

	data, err := dbFetcher()
	if err != nil {
		var zero T
		return zero, err
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pipe := rdb.client.Pipeline()

		jsonData, _ := json.Marshal(data)
		pipe.Set(bgCtx, key, jsonData, expiration)

		for _, dep := range data.GetDependencies() {
			depKey := BuildKeyDep(dep.Domain, dep.Id)
			pipe.SAdd(bgCtx, depKey, key)
			pipe.Expire(bgCtx, depKey, expiration)
		}

		_, _ = pipe.Exec(bgCtx)
	}()

	return data, nil
}

func GetList[T Cacheable](
	ctx context.Context,
	domain string,
	cacheParams map[string]string,
	expiration time.Duration,
	dbFetcher func() ([]T, error),
) ([]T, error) {
	rdb := GetRedisClient()
	listKey := BuildKeyList(domain, cacheParams)

	var cacheHit = false
	var result []T

	val, err := rdb.client.Get(ctx, listKey).Result()

	if err == nil {
		var ids []string
		_ = json.Unmarshal([]byte(val), &ids)

		if len(ids) > 0 {
			var keys []string
			for _, id := range ids {
				keys = append(keys, BuildKeyItem(domain, id))
			}

			itemsJSON, _ := rdb.client.MGet(ctx, keys...).Result()

			var tempResult []T
			allFound := true

			for _, item := range itemsJSON {
				if item == nil {
					allFound = false
					break
				}

				var t T
				if str, ok := item.(string); ok {
					if err := json.Unmarshal([]byte(str), &t); err != nil {
						allFound = false
						break
					}
					tempResult = append(tempResult, t)
				} else {
					allFound = false
					break
				}
			}

			if allFound {
				result = tempResult
				cacheHit = true
			}
		} else {
			return []T{}, nil
		}
	}

	if cacheHit {
		return result, nil
	}

	data, err := dbFetcher()
	if err != nil {
		return nil, err
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pipe := rdb.client.Pipeline()
		var ids []string

		for _, item := range data {
			id := item.GetID()
			ids = append(ids, id)

			mainKey := BuildKeyItem(domain, id)

			jsonData, _ := json.Marshal(item)
			pipe.Set(bgCtx, mainKey, jsonData, expiration+5*time.Minute)

			for _, dep := range item.GetDependencies() {
				depKey := BuildKeyDep(dep.Domain, dep.Id)
				pipe.SAdd(bgCtx, depKey, mainKey)
				pipe.Expire(bgCtx, depKey, expiration)
			}
		}

		idsJSON, _ := json.Marshal(ids)
		pipe.Set(bgCtx, listKey, idsJSON, expiration)

		_, _ = pipe.Exec(bgCtx)
	}()

	return data, nil
}
