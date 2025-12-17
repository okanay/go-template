package redis

import (
	"context"
	"fmt"
	"sync"
)

// InvalidateItem: Bir içeriği siler.
// ÖNEMLİ: Bu fonksiyon hem "app:blog:1"i siler, hem de "app:blog:1:*" taraması yapıp viewleri siler.
func InvalidateItem(ctx context.Context, domain string, id string) error {
	rdb := GetClient()

	// 1. Ana key ve Wildcard pattern
	mainKey := BuildKeyItem(domain, id)             // app:blog:1
	wildcardPattern := BuildWildcardKey(domain, id) // app:blog:1:*

	// Silinecek keyleri topla
	var keysToDelete []string
	keysToDelete = append(keysToDelete, mainKey)

	// SCAN ile wildcard (View'leri bul: card, page vs.)
	// UNLINK (async delete) kullanacağız, o yüzden performansı çok etkilemez ama SCAN count dikkatli seçilmeli.
	iter := rdb.client.Scan(ctx, 0, wildcardPattern, 100).Iterator()
	for iter.Next(ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keysToDelete) > 0 {
		// Del yerine Unlink kullanıyoruz (Non-blocking delete)
		return rdb.client.Unlink(ctx, keysToDelete...).Err()
	}

	return nil
}

// InvalidateByDependency: Bir dependency (örn: Author) değiştiğinde tetiklenir.
// Author:55 değişti -> Ona bağlı Blog:1, Blog:2'yi bul -> Onları InvalidateItem ile (wildcardlı) sil.
func InvalidateByDependency(ctx context.Context, depDomain string, depId string) error {
	rdb := GetClient()

	// Dependency Key: app:deps:author:55
	depKey := BuildKeyDep(depDomain, depId)

	// Bu yazara bağlı olan item key'lerini (örn: app:blog:1, app:blog:2) çek
	dependents, err := rdb.client.SMembers(ctx, depKey).Result()
	if err != nil {
		return err
	}

	if len(dependents) == 0 {
		return nil
	}

	// Her bir bağımlı item için "Wildcard" temizliği yapmamız lazım.
	// Pipeline kullanamıyoruz çünkü her biri için ayrı SCAN yapılması gerekebilir veya
	// key parse edilip domain/id çıkarılmalı.
	// Basitlik adına döngü ile gidiyoruz, zaten Unlink hızlıdır.

	for _, itemKey := range dependents {
		// itemKey örneği: "app:blog:1"
		// Buradan domain ve id'yi ayrıştırmamız lazım ki InvalidateItem çağırabilelim.
		// Ya da manuel scan yaparız. Manuel scan daha güvenli:

		// 1. Ana itemi sil
		rdb.client.Unlink(ctx, itemKey)

		// 2. Varyasyonları (viewleri) sil: itemKey + ":*"
		pattern := itemKey + ":*"
		iter := rdb.client.Scan(ctx, 0, pattern, 50).Iterator()
		var viewKeys []string
		for iter.Next(ctx) {
			viewKeys = append(viewKeys, iter.Val())
		}
		if len(viewKeys) > 0 {
			rdb.client.Unlink(ctx, viewKeys...)
		}
	}

	// Son olarak dependency setini temizle
	rdb.client.Del(ctx, depKey)

	return nil
}

// Tam temizlik yapan helper fonksiyon.
// Senaryo: Bir "Author" güncellendiğinde;
// 1. Author'ın kendi cache'ini sil (InvalidateItem)
// 2. Author'a bağlı olan Blog postlarını bul ve sil (InvalidateByDependency)
func InvalidateEntity(ctx context.Context, domain string, id string) error {
	// Hataları toplamak için basit bir kanal veya sadece ilk hatayı döndüren yapı kurabiliriz.
	// Parallel çalıştırarak işlemi hızlandırıyoruz.

	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	// 1. Kendi cache'ini sil (app:author:1 ve app:author:1:*)
	go func() {
		defer wg.Done()
		if err := InvalidateItem(ctx, domain, id); err != nil {
			errChan <- fmt.Errorf("item invalidation failed: %w", err)
		}
	}()

	// 2. Buna bağımlı olanları sil (app:deps:author:1 listesindekiler)
	go func() {
		defer wg.Done()
		if err := InvalidateByDependency(ctx, domain, id); err != nil {
			errChan <- fmt.Errorf("dependency invalidation failed: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// Kanalda hata var mı kontrol et
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// Seçili olan DB'deki (örneğin DB 0) TÜM veriyi siler.
// DİKKAT: Production ortamında asla çağrılmamalıdır.
func InvalidateFlushDB() error {
	rdb := GetClient()
	return rdb.client.FlushDB(context.Background()).Err()
}
