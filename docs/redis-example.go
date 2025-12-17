package redis_example

import (
	"context"
	"encoding/json"
	"time"
)

func (r *CachedRepository) SelectRedisOrPSQL(ctx context.Context, category string) ([]FileEntity, error) {
	listKey := "file:list:cat:" + category

	// 1. ADIM: Redis Kontrolü: Listeyi Getir (Sadece ID'ler)
	val, err := r.redis.Get(ctx, listKey).Result()

	// --- SENARYO A: LİSTE REDİS'TE VAR (CACHE HIT) ---
	if err == nil {
		var ids []string
		json.Unmarshal([]byte(val), &ids)

		if len(ids) > 0 {
			// ID'leri kullanarak item'leri çek (MGET)
			var keys []string
			for _, id := range ids {
				keys = append(keys, "file:item:"+id)
			}

			// Tek seferde bütün detayları çek
			itemsJSON, _ := r.redis.MGet(ctx, keys...)

			var files []FileEntity
			for _, item := range itemsJSON {
				if item == nil {
					// HATA: Listede ID var ama Data yok (Nadir durum)
					// Burada DB'ye gidebiliriz (Hydration)
					// Fakat en doğru yer burası değil gibi 2-3 nill eleman varsa her seferinde veritabanına mı gideceğiz?
					// Nil olanları bir listeye alıp en son toplu get yapmak lazım.
					// Daha sonra onları redise kayıt etmemiz lazım.
					continue
				}
				var f FileEntity
				json.Unmarshal([]byte(item.(string)), &f)
				files = append(files, f)
			}
			return files, nil
		}
		// Liste boşsa boş array dön
		return []FileEntity{}, nil
	}

	// --- SENARYO B: LİSTE REDİS'TE YOK (CACHE MISS) ---

	// 2. ADIM: PSQL'e git ve datayı oluştur
	files, err := r.next.SelectFilesByCategory(ctx, category) // DB Sorgusu
	if err != nil {
		return nil, err
	}

	// Response'u hemen kullanıcıya dönmeliyiz
	// 3. ADIM: Arka planda Redis'i doldur.
	go func() {
		// DİKKAT: Burada HTTP Context'i kullanmamız sakıncalı, HTTP context çoktan gebermiş olabilir. Yeni context oluştur.
		bgCtx := context.Background()

		var ids []string
		pipe := r.redis.Pipeline()
		// NOTE :: PIPE bilmediğim bir konsept bunu Gemini 3.0 önerdi.
		// TODO :: Redis PIPE ne işe yarıyor araştır.

		for _, f := range files {
			// a. ID listesi için ID'yi topla
			// ID String, Number veya UUID olabilir. Burada bussines logic ile durumu yönet.
			ids = append(ids, f.ID)

			// b. Elemanları tek tek PIPE'a ekle. (file:item:$id)
			data, _ := json.Marshal(f)
			pipe.Set(bgCtx, "file:item:"+f.ID, data, 1*time.Hour)
		}

		// c. Listeyi []byte haline getir ve PIPE'a ekle (file:list:$cat)
		idsJSON, _ := json.Marshal(ids)
		pipe.Set(bgCtx, listKey, idsJSON, 1*time.Hour)

		// Hepsini tek seferde Redis'e gönder
		_, _ = pipe.Exec(bgCtx)
	}()

	return files, nil
}

// ************---------******************
// IGNORE << DUMMY TYPE >> IGNORE
// ************---------******************

type FileEntity struct {
	ID       string
	Category string
	Name     string
	Size     int64
}

type CachedRepository struct {
	redis RedisClient
	next  FileRepository
}

type RedisClient interface {
	Get(ctx context.Context, key string) *StringCmd
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *StatusCmd
	Pipeline() Pipeliner
}

type FileRepository interface {
	SelectFilesByCategory(ctx context.Context, category string) ([]FileEntity, error)
}

type StringCmd struct {
	val string
	err error
}

func (c *StringCmd) Result() (string, error) {
	return c.val, c.err
}

type StatusCmd struct {
	err error
}

func (c *StatusCmd) Err() error {
	return c.err
}

type Pipeliner interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration)
	Exec(ctx context.Context) ([]Cmder, error)
}

type Cmder interface {
	Err() error
}
