package redis

import (
	"context"
	"encoding/json"
	"time"
)

// UpsertItem: Veritabanında güncelleme veya ekleme yapıldıktan sonra çağrılır.
// Cache'teki item'ı anında günceller (Write-Through benzeri).
// Varsa üzerine yazar, yoksa oluşturur.
func UpsertItem[T Cacheable](
	ctx context.Context,
	domain string,
	data T,
	expiration time.Duration,
) error {
	rdb := GetRedisClient()
	// 1. Key Oluştur
	id := data.GetID()
	key := BuildKeyItem(domain, id)

	// 2. Data'yı JSON yap
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 3. Pipeline Başlat (Atomik işlem için)
	pipe := rdb.client.Pipeline()

	// A. Ana datayı yaz (Varsa ezer, yoksa oluşturur - UPSERT)
	pipe.Set(ctx, key, jsonData, expiration)

	// B. Dependency (Bağımlılık) güncellemeleri
	// Örneğin Blog güncellendiğinde, Author ID değişmiş olabilir.
	// Yeni dependency setine de eklemeliyiz.
	for _, dep := range data.GetDependencies() {
		depKey := BuildKeyDep(dep.Domain, dep.Id)
		pipe.SAdd(ctx, depKey, key)
		pipe.Expire(ctx, depKey, expiration)
	}

	// Not: Eski dependency'i silmeye çalışmıyoruz (Karmaşıklık yaratır).
	// Redis'te fazladan dependency kalması sisteme zarar vermez,
	// sadece fazladan bir invalidate tetikler ki bu güvenlidir.

	_, err = pipe.Exec(ctx)
	return err
}
