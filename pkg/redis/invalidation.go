package redis

import (
	"context"
	"fmt"
)

// Bir entity güncellendiğinde veya silindiğinde çağrılır.
// Hem kendi item cache'ini siler, hem de ona bağlı olan listeleri (opsiyonel) temizler.
func InvalidateItem(ctx context.Context, rdb *RedisClient, domain string, id string) error {
	key := rdb.BuildKeyItem(domain, id)
	return rdb.client.Del(ctx, key).Err()
}

// Bir dependency (örn: Author) değiştiğinde,
// ona bağımlı olan asıl kayıtları (örn: Blogları) bulur ve siler.
func InvalidateByDependency(ctx context.Context, rdb *RedisClient, domain string, id string) error {
	// Örn: depString = "user:123" -> Key: "deps:user:123"
	depKey := rdb.BuildKeyDep(domain, id)

	// 1. Bu dependency'e bağlı olan item key'lerini (örn: blog:item:100, blog:item:101) çek
	keys, err := rdb.client.SMembers(ctx, depKey).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// 2. Pipeline ile hepsini tek seferde sil
	pipe := rdb.client.Pipeline()

	// Bağımlı itemları sil (blogları uçur)
	pipe.Del(ctx, keys...)

	// Dependency set'ini de temizle (artık gerek kalmadı, bloglar tekrar okununca set yine oluşacak)
	pipe.Del(ctx, depKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("dependency invalidation failed: %w", err)
	}

	return nil
}

// Bir kayıt silindiğinde/güncellendiğinde çağır.
// Hem kaydın kendisini siler, hem de ona bağımlı olanları (örn: Yazarın blogları) temizler.
func InvalidateItemWithDep(ctx context.Context, rdb *RedisClient, domain string, id string) error {
	// 1. Silinecek anahtarları topla
	keysToDelete := []string{}

	// Ana item key'i (örn: user:item:123)
	itemKey := rdb.BuildKeyItem(domain, id)
	keysToDelete = append(keysToDelete, itemKey)

	depKey := rdb.BuildKeyDep(domain, id)

	// Bu kayda bağımlı olan diğer kayıtları bul (örn: blog:item:10, blog:item:11)
	dependents, _ := rdb.client.SMembers(ctx, depKey).Result()
	if len(dependents) > 0 {
		keysToDelete = append(keysToDelete, dependents...)
	}

	// 2. Hepsini tek seferde sil (Pipeline ile atomik işlem)
	pipe := rdb.client.Pipeline()
	pipe.Del(ctx, keysToDelete...) // Itemlar ve bağımlıları sil
	pipe.Del(ctx, depKey)          // Dependency listesini de sil

	_, err := pipe.Exec(ctx)
	return err
}

// Örn: "blog:list:*" dersen tüm blog listelerini temizler.
func InvalidatePattern(ctx context.Context, rdb *RedisClient, pattern string) error {
	// SCAN komutu Redis'i kilitlemeden arama yapar.
	iter := rdb.client.Scan(ctx, 0, pattern, 0).Iterator()

	// Bulduğu keyleri biriktirip silmek için
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	// Eğer silinecek bir şey bulduysa sil
	if len(keys) > 0 {
		// UNLINK, DEL komutunun daha hızlısı ve asenkron olanıdır.
		// Büyük listeleri silerken Redis'i yormaz.
		return rdb.client.Unlink(ctx, keys...).Err()
	}

	return nil
}

func FlushCache(ctx context.Context, rdb *RedisClient) error {
	return rdb.client.FlushDB(ctx).Err()
}
