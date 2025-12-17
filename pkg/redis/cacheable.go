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

// Tek bir item'ı cache'ten veya DB'den getirir.
func GetItem[T Cacheable](
	ctx context.Context,
	rdb *RedisClient,
	domain string,
	id string,
	expiration time.Duration,
	dbFetcher func() (T, error),
) (T, error) {
	// 1. Key Oluşturma
	key := rdb.BuildKeyItem(domain, id)

	var result T
	var cacheHit = false

	// 2. Redis'ten Sorgula
	val, err := rdb.client.Get(ctx, key).Result()

	if err == nil {
		// Cache Hit! JSON'ı parse et
		if jsonErr := json.Unmarshal([]byte(val), &result); jsonErr == nil {
			cacheHit = true
		}
		// Eğer JSON bozuksa (jsonErr != nil), cacheHit false kalır, aşağıdan DB'ye gideriz (Auto-heal).
	} else if err != redis.Nil {
		// redis.Nil dışındaki hatalar (bağlantı kopması vs.) loglanabilir
		// ama akışı bozmamak için DB'ye gitmeye devam ederiz.
	}

	if cacheHit {
		return result, nil
	}

	// 3. Cache Miss: DB'den çek
	data, err := dbFetcher()
	if err != nil {
		// Data DB'de de yoksa veya DB hatası varsa boş değer ve hatayı dön
		var zero T
		return zero, err
	}

	// 4. Redis'e Yaz (Asenkron - Response süresini uzatmamak için)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pipe := rdb.client.Pipeline()

		// A. Datayı JSON yap
		jsonData, _ := json.Marshal(data)
		pipe.Set(bgCtx, key, jsonData, expiration)

		// B. Dependency (Bağımlılık) Kayıtlarını Oluştur
		// Bu kısım ÇOK ÖNEMLİ. Detay sayfası ilk kez açıldığında da
		// "Bu blog şu yazara aittir" bilgisini Redis'e işlemeliyiz.
		for _, dep := range data.GetDependencies() {
			depKey := rdb.BuildKeyDep(dep.Domain, dep.Id)
			// Set'e ekle: "deps:user:5" setine "blog:item:100" eklenir.
			pipe.SAdd(bgCtx, depKey, key)
			// Dependency key'in ömrünü de uzat (veya sabitle)
			pipe.Expire(bgCtx, depKey, expiration)
		}

		_, _ = pipe.Exec(bgCtx)
	}()

	return data, nil
}

// Cache'lenmiş bir listeyi getirir veya oluşturur.
func GetList[T Cacheable](
	ctx context.Context,
	rdb *RedisClient,
	domain string,
	cacheParams map[string]string,
	expiration time.Duration,
	dbFetcher func() ([]T, error),
) ([]T, error) {
	listKey := rdb.BuildKeyList(domain, cacheParams)

	// Cache Hit durumunu takip edecek flag
	var cacheHit = false
	var result []T

	// 2. ADIM: Redis List Key Kontrolü
	val, err := rdb.client.Get(ctx, listKey).Result()

	if err == nil {
		var ids []string
		_ = json.Unmarshal([]byte(val), &ids)

		if len(ids) > 0 {
			var keys []string
			for _, id := range ids {
				// BURASI DEĞİŞTİ: Item key'ini Builder üretiyor
				keys = append(keys, rdb.BuildKeyItem(domain, id))
			}

			// MGET operasyonları aynı kalır...
			itemsJSON, _ := rdb.client.MGet(ctx, keys...).Result()

			// Geçici slice, eğer hepsi varsa result'a atayacağız
			var tempResult []T
			allFound := true // Varsayalım ki hepsi bulundu

			for _, item := range itemsJSON {
				// Eğer tek bir item bile nil ise, döngüyü kır.
				if item == nil {
					allFound = false
					break
				}

				var t T
				if str, ok := item.(string); ok {
					if err := json.Unmarshal([]byte(str), &t); err != nil {
						// JSON bozuksa da cache bozuk demektir
						allFound = false
						break
					}
					tempResult = append(tempResult, t)
				} else {
					allFound = false
					break
				}
			}

			// Sadece hepsi sorunsuz bulunduysa Cache Hit kabul et
			if allFound {
				result = tempResult
				cacheHit = true
			}
		} else {
			// Liste var ama boş ([]), bu da bir hit sayılır
			return []T{}, nil
		}
	}

	// --- SENARYO A: TAM CACHE HIT ---
	if cacheHit {
		return result, nil
	}

	// --- SENARYO B: CACHE MISS veya PARTIAL MISS (Eksik Data) ---
	// Kod buraya geldiyse ya key yok ya da data eksik/bozuk.
	// DB'ye gidip doğrusunu alalım ve Redis'i onaralım.

	data, err := dbFetcher()
	if err != nil {
		return nil, err
	}

	// 3. ADIM: Arka planda Redis'i doldur/onar (Pipeline)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pipe := rdb.client.Pipeline()
		var ids []string

		for _, item := range data {
			id := item.GetID()
			ids = append(ids, id)

			// Builder Kullanımı: Item Key
			mainKey := rdb.BuildKeyItem(domain, id)

			jsonData, _ := json.Marshal(item)
			pipe.Set(bgCtx, mainKey, jsonData, expiration+5*time.Minute)

			// Builder Kullanımı: Dependencies
			for _, dep := range item.GetDependencies() {
				depKey := rdb.BuildKeyDep(dep.Domain, dep.Id)
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
