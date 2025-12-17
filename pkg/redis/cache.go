package redis

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

// Cacheable arayüzü
type Cacheable interface {
	GetID() string
	GetDependencies() []Dependency
}

type Dependency struct {
	Domain string
	ID     string
}

// GetOptions view gibi opsiyonel parametreler için
type GetOptions struct {
	View string // boşsa ana item, doluysa "card", "summary" vs.
}

func GetItem[T Cacheable](
	ctx context.Context,
	domain string,
	id string,
	expiration time.Duration,
	opts GetOptions,
	dbFetcher func() (T, error),
) (T, error) {
	rdb := GetClient()

	// View varsa key değişir: app:blog:1:view:card
	// Yoksa ana key: app:blog:1
	var key string
	if opts.View != "" {
		key = BuildKeyView(domain, id, opts.View)
	} else {
		key = BuildKeyItem(domain, id)
	}

	var result T

	// 1. Redis'ten çekmeyi dene
	val, err := rdb.client.Get(ctx, key).Result()
	if err == nil {
		if jsonErr := json.Unmarshal([]byte(val), &result); jsonErr == nil {
			return result, nil // Cache Hit
		}
	} else if err != redis.Nil {
		// Redis hatası (bağlantı vs), loglanabilir ama akışı bozmayalım
	}

	// 2. Cache Miss -> DB'den çek
	data, err := dbFetcher()
	if err != nil {
		var zero T
		return zero, err
	}

	// 3. Asenkron olarak Redis'e yaz (Cache Aside)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pipe := rdb.client.Pipeline()

		jsonData, _ := json.Marshal(data)
		pipe.Set(bgCtx, key, jsonData, expiration)

		// Bağımlılıkları kaydet
		// Eğer bu bir "View" ise bile, ana item ID'si üzerinden dependency kurulmalı
		// Örnek: Author değişirse -> Blog:1 silinmeli (Blog:1 silinince view'leri de taranıp silinecek)

		// Sadece ana item kaydedilirken dependency set etmek genellikle yeterlidir,
		// ama view'ler için de garanti olsun diye ana item key'ini dependency'e ekliyoruz.
		// NOT: Dependency listesine "app:blog:1" ekliyoruz (view uzantısı olmadan).
		// Çünkü invalidate ederken wildcard taraması yapacağız.
		baseItemKey := BuildKeyItem(domain, id)

		for _, dep := range data.GetDependencies() {
			depKey := BuildKeyDep(dep.Domain, dep.ID)
			pipe.SAdd(bgCtx, depKey, baseItemKey)
			pipe.Expire(bgCtx, depKey, expiration)
		}

		_, _ = pipe.Exec(bgCtx)
	}()

	return data, nil
}

// GetList: Belirli bir kriterdeki listeyi ve elemanlarını çeker.
func GetList[T Cacheable](
	ctx context.Context,
	domain string,
	cacheParams map[string]string, // Sort, filter parametreleri
	expiration time.Duration,
	opts GetOptions, // View bilgisi burada (örn: "card")
	dbFetcher func() ([]T, error),
) ([]T, error) {
	rdb := GetClient()

	// Liste Key'i: app:blog:list:page=1...
	listKey := BuildKeyList(domain, cacheParams)

	var cacheHit = false
	var result []T

	// 1. Önce listenin kendisini (ID listesini) çek
	val, err := rdb.client.Get(ctx, listKey).Result()

	if err == nil {
		var ids []string
		_ = json.Unmarshal([]byte(val), &ids)

		if len(ids) > 0 {
			// ID'leri bulduk, şimdi item'ların kendisini çekmeliyiz (MGET)
			var keys []string
			for _, id := range ids {
				// Eğer View="card" ise key: app:blog:1:view:card olur
				if opts.View != "" {
					keys = append(keys, BuildKeyView(domain, id, opts.View))
				} else {
					keys = append(keys, BuildKeyItem(domain, id))
				}
			}

			// MGET ile toplu çekim
			itemsJSON, errMget := rdb.client.MGet(ctx, keys...).Result()

			if errMget == nil {
				var tempResult []T
				allFound := true

				for _, item := range itemsJSON {
					// MGET'te bir key bile yoksa nil döner
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
			}
		} else {
			// Liste boş olarak cache'lenmiş olabilir
			return []T{}, nil
		}
	}

	if cacheHit {
		return result, nil
	}

	// 2. Cache Miss -> DB'den çek
	data, err := dbFetcher()
	if err != nil {
		return nil, err
	}

	// 3. Asenkron Kayıt (Cache Aside)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Liste yazma uzun sürebilir
		defer cancel()

		pipe := rdb.client.Pipeline()
		var ids []string

		for _, item := range data {
			id := item.GetID()
			ids = append(ids, id)

			// Key belirle (View varsa view key, yoksa normal key)
			var itemKey string
			if opts.View != "" {
				itemKey = BuildKeyView(domain, id, opts.View)
			} else {
				itemKey = BuildKeyItem(domain, id)
			}

			jsonData, _ := json.Marshal(item)
			// Item'ları 5dk daha uzun tutuyoruz ki liste expire olsa bile item cache'den gelebilsin
			pipe.Set(bgCtx, itemKey, jsonData, expiration+5*time.Minute)

			// Dependency yönetimi:
			// View olsa bile dependency ana item ID üzerine kurulur (app:blog:1)
			// Çünkü invalidate ederken wildcard (app:blog:1*) kullanıyoruz.
			baseItemKey := BuildKeyItem(domain, id)

			for _, dep := range item.GetDependencies() {
				depKey := BuildKeyDep(dep.Domain, dep.ID)
				pipe.SAdd(bgCtx, depKey, baseItemKey)
				pipe.Expire(bgCtx, depKey, expiration)
			}
		}

		// ID Listesini kaydet
		idsJSON, _ := json.Marshal(ids)
		pipe.Set(bgCtx, listKey, idsJSON, expiration)

		_, _ = pipe.Exec(bgCtx)
	}()

	return data, nil
}

// Bir veriyi cache'e yazar veya günceller.
// Genellikle DB'ye yazma işleminden hemen sonra cache'i tazelemek için kullanılır.
func UpsertItem[T Cacheable](
	ctx context.Context,
	domain string,
	data T,
	expiration time.Duration,
	opts GetOptions, // Eğer bir View (örn: "card") kaydediyorsan belirtmelisin
) error {
	rdb := GetClient()
	id := data.GetID()

	// 1. Hangi Key'e yazacağız? (Main veya View)
	var key string
	if opts.View != "" {
		key = BuildKeyView(domain, id, opts.View)
	} else {
		key = BuildKeyItem(domain, id)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Pipeline başlat (Atomik olmasa da network round-trip azaltır)
	pipe := rdb.client.Pipeline()

	// Datayı yaz
	pipe.Set(ctx, key, jsonData, expiration)

	// 2. Dependency Yönetimi
	// KRİTİK NOKTA: View kaydediyor olsak bile, dependency'ye ANA KEY'i (baseItemKey) ekliyoruz.
	// Çünkü InvalidateByDependency çalıştığında "app:blog:1" bulur ve wildcard ile "app:blog:1:*" siler.
	baseItemKey := BuildKeyItem(domain, id)

	for _, dep := range data.GetDependencies() {
		depKey := BuildKeyDep(dep.Domain, dep.ID)
		pipe.SAdd(ctx, depKey, baseItemKey)
		pipe.Expire(ctx, depKey, expiration)
	}

	_, err = pipe.Exec(ctx)
	return err
}
