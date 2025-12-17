# Redis Cache-Aside Package Documentation

Bu paket, uygulamanın **Cache-Aside (Önbellek Kenara)** stratejisini yönetir. Redis burada "Source of Truth" (Gerçek Veri Kaynağı) değil, veritabanını rahatlatan bir **hızlandırıcı** görevi görür.

---

## 1. Temel Felsefe

### Cache-Aside Nedir?

```
┌─────────┐      1. GET key      ┌─────────┐
│   App   │ ──────────────────►  │  Redis  │
│         │ ◄──────────────────  │         │
└────┬────┘      2. HIT/MISS     └─────────┘
     │
     │ 3. MISS ise
     ▼
┌─────────┐
│   DB    │  ◄── Veriyi çek
└─────────┘
     │
     │ 4. Kullanıcıya dön
     │ 5. Async olarak Redis'e yaz (goroutine)
     ▼
```

**Önemli Kararlar:**

| Karar | Açıklama |
|-------|----------|
| **Eventual Consistency** | Cache yazma işlemi arka planda (`go func`) yapılır. Kullanıcı DB'den taze veriyi anında alır, Redis milisaniyeler sonra güncellenir. |
| **View Granülaritesi** | Bir içeriğin ham hali ile "Card" veya "Summary" görünümleri farklı cache anahtarlarında tutulur ama hepsi tek bir kaynağa bağlıdır. |
| **Non-blocking Delete** | `DEL` yerine `UNLINK` kullanılır. Büyük verilerde Redis donmaz. |

---

## 2. Key Stratejisi (Anahtar Yapısı)

Redis'in ağaç yapısı yoktur, düz (flat) bir yapıdır. Hiyerarşiyi simüle etmek için isimlendirme standardı kullanıyoruz.

### Key Tipleri ve Örnekleri

```
┌─────────────────────────────────────────────────────────────────┐
│                        KEY HİYERARŞİSİ                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  app:blog:101                    ◄── Base Item (Ham veri)       │
│    │                                                            │
│    ├── app:blog:101:view:card    ◄── Card görünümü              │
│    ├── app:blog:101:view:summary ◄── Summary görünümü           │
│    └── app:blog:101:view:rss     ◄── RSS görünümü               │
│                                                                 │
│  app:deps:author:55              ◄── Dependency Set             │
│    │                                 (İçinde: app:blog:101,     │
│    │                                  app:blog:102, ...)        │
│                                                                 │
│  app:blog:list:page=1:sort=desc  ◄── Liste (Sadece ID'ler)      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Builder Fonksiyonları

```go
// keys.go içindeki fonksiyonlar

BuildKeyItem("blog", "101")           // → app:blog:101
BuildKeyView("blog", "101", "card")   // → app:blog:101:view:card
BuildWildcardKey("blog", "101")       // → app:blog:101:*
BuildKeyDep("author", "55")           // → app:deps:author:55
BuildKeyList("blog", map[string]string{
    "page": "1",
    "sort": "desc",
})                                    // → app:blog:list:page=1:sort=desc
```

---

## 3. Akış Diyagramları

### A. Veri Okuma: `GetItem`

```
Senaryo: Kullanıcı Blog #101'in Card görünümünü istiyor

┌──────────────────────────────────────────────────────────────────┐
│                         GetItem AKIŞI                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  User Request                                                    │
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────────────────────────────────┐                     │
│  │ Key Oluştur:                            │                     │
│  │ opts.View="card" → app:blog:101:view:card│                    │
│  └────────────────────┬────────────────────┘                     │
│                       │                                          │
│                       ▼                                          │
│              ┌────────────────┐                                  │
│              │  Redis GET     │                                  │
│              └───────┬────────┘                                  │
│                      │                                           │
│           ┌──────────┴──────────┐                                │
│           │                     │                                │
│      [HIT ✓]               [MISS ✗]                              │
│           │                     │                                │
│           ▼                     ▼                                │
│    JSON Unmarshal         DB'den Çek                             │
│           │                     │                                │
│           │                     ├──► Kullanıcıya Dön             │
│           │                     │                                │
│           │                     └──► go func() {                 │
│           │                           • SET app:blog:101:view:card
│           │                           • SADD app:deps:author:55  │
│           │                                  → app:blog:101      │
│           │                         }                            │
│           │                                                      │
│           ▼                                                      │
│    Return Data                                                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Kod Örneği:**

```go
blog, err := redis.GetItem[BlogPost](
    ctx,
    "blog",                    // domain
    "101",                     // id
    30*time.Minute,            // expiration
    redis.GetOptions{View: "card"},
    func() (BlogPost, error) { // dbFetcher
        return db.GetBlogByID(101)
    },
)
```

### B. Liste Okuma: `GetList`

```
┌──────────────────────────────────────────────────────────────────┐
│                         GetList AKIŞI                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Liste Key'ini Çek                                            │
│     GET app:blog:list:page=1:sort=desc                           │
│           │                                                      │
│           ▼                                                      │
│     ["101", "102", "103"]  ◄── Sadece ID'ler                     │
│           │                                                      │
│           │                                                      │
│  2. ID'lerden Item Key'leri Oluştur (View varsa view key)        │
│           │                                                      │
│           ▼                                                      │
│     [                                                            │
│       "app:blog:101:view:card",                                  │
│       "app:blog:102:view:card",                                  │
│       "app:blog:103:view:card"                                   │
│     ]                                                            │
│           │                                                      │
│           │                                                      │
│  3. MGET ile Toplu Çekim                                         │
│           │                                                      │
│           ▼                                                      │
│     ┌─────────────────────────────────────┐                      │
│     │ Tüm item'lar bulundu mu?            │                      │
│     └──────────────┬──────────────────────┘                      │
│                    │                                             │
│         ┌──────────┴──────────┐                                  │
│         │                     │                                  │
│    [EVET ✓]              [HAYIR ✗]                               │
│         │                     │                                  │
│         ▼                     ▼                                  │
│   Return Data           DB'den Çek                               │
│                               │                                  │
│                               └──► Async Cache Yaz               │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Neden ID Listesi + MGET?**

```
❌ Yanlış Yaklaşım: Tüm listeyi tek key'de tutmak
   - 1000 blog postlu liste = Çok büyük JSON
   - Bir post güncellenince tüm liste invalidate

✅ Doğru Yaklaşım: ID Referansı + MGET
   - Liste key'i sadece ID tutar (küçük)
   - Item'lar ayrı key'lerde (granüler invalidation)
   - MGET ile tek roundtrip'te hepsi çekilir
```

---

## 4. Wildcard Temizliği (En Kritik Kısım)

### Problem: Redis'te Cascading Delete Yok

```
Redis'te DEL app:blog:101 komutu SADECE o key'i siler!

app:blog:101           ◄── Silindi ✓
app:blog:101:view:card ◄── Hâlâ duruyor! ✗
app:blog:101:view:rss  ◄── Hâlâ duruyor! ✗
```

### Çözüm: SCAN + UNLINK Pattern'i

```go
// invalidation.go - InvalidateItem fonksiyonu

func InvalidateItem(ctx context.Context, domain string, id string) error {
    rdb := GetClient()

    // 1. Ana key
    mainKey := BuildKeyItem(domain, id)             // app:blog:101

    // 2. Wildcard pattern
    wildcardPattern := BuildWildcardKey(domain, id) // app:blog:101:*

    var keysToDelete []string
    keysToDelete = append(keysToDelete, mainKey)

    // 3. SCAN ile view'leri bul
    iter := rdb.client.Scan(ctx, 0, wildcardPattern, 100).Iterator()
    for iter.Next(ctx) {
        keysToDelete = append(keysToDelete, iter.Val())
    }

    // 4. UNLINK ile async sil (DEL değil!)
    if len(keysToDelete) > 0 {
        return rdb.client.Unlink(ctx, keysToDelete...).Err()
    }

    return nil
}
```

**Görsel Akış:**

```
InvalidateItem("blog", "101") çağrıldığında:

┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Step 1: Key'leri Topla                                         │
│  ──────────────────────                                         │
│                                                                 │
│  mainKey = "app:blog:101"                                       │
│                                                                 │
│  SCAN 0 MATCH "app:blog:101:*" COUNT 100                        │
│    │                                                            │
│    ├── app:blog:101:view:card                                   │
│    ├── app:blog:101:view:summary                                │
│    └── app:blog:101:view:rss                                    │
│                                                                 │
│  keysToDelete = [                                               │
│    "app:blog:101",                                              │
│    "app:blog:101:view:card",                                    │
│    "app:blog:101:view:summary",                                 │
│    "app:blog:101:view:rss"                                      │
│  ]                                                              │
│                                                                 │
│  Step 2: Toplu Sil                                              │
│  ─────────────────                                              │
│                                                                 │
│  UNLINK app:blog:101 app:blog:101:view:card ...                 │
│         ▲                                                       │
│         │                                                       │
│         └── Non-blocking! Redis donmaz.                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Dependency (Bağımlılık) Sistemi

### Senaryo: Author Güncellendiğinde Ne Olur?

```
Author #55 "Ahmet Yılmaz" adını "Ahmet Can" olarak değiştirdi.

Bu yazara ait:
  - Blog #101 (Card view'da yazar adı gösteriliyor)
  - Blog #102
  - Blog #103

Hepsinin cache'i geçersiz olmalı!
```

### Dependency Kaydı Nasıl Yapılır?

```go
// Blog modeli Cacheable interface'i implemente eder

type BlogPost struct {
    ID       string
    Title    string
    AuthorID string
    // ...
}

func (b BlogPost) GetID() string {
    return b.ID
}

func (b BlogPost) GetDependencies() []redis.Dependency {
    return []redis.Dependency{
        {Domain: "author", ID: b.AuthorID},
        // Kategori bağımlılığı da eklenebilir:
        // {Domain: "category", ID: b.CategoryID},
    }
}
```

**Cache yazılırken (GetItem/GetList içinde):**

```
Blog #101 cache'lenirken:

SET app:blog:101:view:card {...}

SADD app:deps:author:55 "app:blog:101"
      ▲                        ▲
      │                        │
      │                        └── Ana key (view değil!)
      │
      └── Dependency Set (Author 55'e bağımlı olanlar)
```

### Dependency Invalidation Akışı

```
┌──────────────────────────────────────────────────────────────────┐
│            InvalidateByDependency("author", "55")                │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Step 1: Bağımlı Item'ları Bul                                   │
│  ─────────────────────────────                                   │
│                                                                  │
│  SMEMBERS app:deps:author:55                                     │
│    │                                                             │
│    ├── "app:blog:101"                                            │
│    ├── "app:blog:102"                                            │
│    └── "app:blog:103"                                            │
│                                                                  │
│  Step 2: Her Biri İçin Wildcard Temizlik                         │
│  ───────────────────────────────────────                         │
│                                                                  │
│  FOR EACH itemKey IN dependents:                                 │
│    │                                                             │
│    ├── UNLINK app:blog:101                                       │
│    │   SCAN app:blog:101:* → UNLINK (card, summary, rss...)      │
│    │                                                             │
│    ├── UNLINK app:blog:102                                       │
│    │   SCAN app:blog:102:* → UNLINK                              │
│    │                                                             │
│    └── UNLINK app:blog:103                                       │
│        SCAN app:blog:103:* → UNLINK                              │
│                                                                  │
│  Step 3: Dependency Set'i Temizle                                │
│  ────────────────────────────────                                │
│                                                                  │
│  DEL app:deps:author:55                                          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### InvalidateEntity: Tam Temizlik

```go
// Author güncellendiğinde tek fonksiyon çağrısı yeterli:

err := redis.InvalidateEntity(ctx, "author", "55")

// Bu fonksiyon PARALEL olarak:
// 1. Author'ın kendi cache'ini siler (app:author:55 ve app:author:55:*)
// 2. Ona bağlı blog postlarını bulur ve wildcard ile siler
```

**Görsel:**

```
InvalidateEntity("author", "55")
           │
           ├──────────────────────────────────────┐
           │                                      │
           ▼                                      ▼
   [Goroutine 1]                         [Goroutine 2]
   InvalidateItem                     InvalidateByDependency
           │                                      │
           ▼                                      ▼
   DEL app:author:55                 SMEMBERS app:deps:author:55
   SCAN app:author:55:*                        │
   UNLINK (varsa view'ler)           ┌─────────┴─────────┐
                                     ▼         ▼         ▼
                                 Blog:101  Blog:102  Blog:103
                                 + Views   + Views   + Views
```

---

## 6. Kritik Notlar ve Best Practices

### UNLINK vs DEL

```
DEL komutu:
  - Senkron çalışır
  - Büyük value'larda Redis'i bloklar
  - 100MB'lık bir key silmek saniyeler alabilir

UNLINK komutu:
  - Asenkron çalışır
  - Key hemen "görünmez" olur
  - Gerçek silme arka planda yapılır
  - Production için GÜVENLİ
```

### Dependency Yönü (Tek Yönlü!)

```
Author değişti → Blog silinir ✓
Blog değişti   → Author SİLİNMEZ ✓

Bağımlılık her zaman Child → Parent yönündedir.
Parent değiştiğinde Child'lar etkilenir, tersi değil.
```

### Singleflight Uyarısı

```
⚠️ Bu pakette "Thundering Herd" koruması YOKTUR!

Senaryo:
  - Popular bir içerik cache'de yok
  - 1000 kullanıcı aynı anda istiyor
  - 1000 ayrı DB sorgusu yapılır! 💥

Çözüm: Service katmanında singleflight kullanın:

import "golang.org/x/sync/singleflight"

var sf singleflight.Group

func GetBlogCached(id string) (BlogPost, error) {
    result, err, _ := sf.Do("blog:"+id, func() (interface{}, error) {
        return redis.GetItem[BlogPost](...)
    })
    return result.(BlogPost), err
}
```

### Dependency Set Boyutu Hakkında

```
Dependency set'inde kaç item olur?

Set'e eklenen item'lar = Cache MISS sonrası async yazılan item'lar

Yani bir item'ın set'te olması için:
  1. TTL süresi içinde okunmuş olmalı
  2. Henüz expire olmamış olmalı

Pratikte:
  ┌─────────────────────────────────────────────────────────┐
  │ Author'ın 500 blog postu var                            │
  │ TTL: 30 dakika                                          │
  │                                                         │
  │ 30dk içinde kaçı okunur? → Belki 50-100                 │
  │ Geri kalanı zaten cache'te yok                          │
  │                                                         │
  │ Dependency set boyutu: ~50-100 entry (binlerce değil!)  │
  │ SCAN maliyeti: 10-50ms (kabul edilebilir)               │
  └─────────────────────────────────────────────────────────┘

NOT: Set'te expire olmuş item key'leri kalabilir.
     UNLINK zaten olmayan key için no-op çalışır (zararsız).
```

### Liste Invalidation

```
⚠️ Item silindiğinde liste key'leri OTOMATİK SİLİNMEZ!

Örnek:
  - app:blog:list:page=1 = ["101", "102", "103"]
  - Blog 102 silindi
  - Liste hâlâ eski ID'yi gösteriyor!

Çözüm Seçenekleri:
  1. Liste key'lerini de invalidate et (önerilen)
  2. GetList'te MGET sonrası nil kontrolü yap (mevcut davranış)
  3. Liste TTL'ini kısa tut
```

---

## 7. Kullanım Örnekleri

### Örnek 1: Tekil Item Okuma

```go
// Bir blog postunun card görünümünü al
post, err := redis.GetItem[BlogPost](
    ctx,
    "blog",
    "101",
    30*time.Minute,
    redis.GetOptions{View: "card"},
    func() (BlogPost, error) {
        // DB'den sadece card için gereken alanları çek
        return repo.GetBlogCardByID(ctx, 101)
    },
)
```

### Örnek 2: Liste Okuma

```go
// Sayfalanmış blog listesi
posts, err := redis.GetList[BlogPost](
    ctx,
    "blog",
    map[string]string{
        "page":   "1",
        "limit":  "20",
        "sort":   "created_at",
        "order":  "desc",
    },
    15*time.Minute,
    redis.GetOptions{View: "card"},
    func() ([]BlogPost, error) {
        return repo.GetBlogList(ctx, page, limit, sort, order)
    },
)
```

### Örnek 3: Güncelleme Sonrası Cache Invalidation

```go
// Blog güncellendi
err := repo.UpdateBlog(ctx, blog)
if err != nil {
    return err
}

// Cache'i temizle (wildcard ile tüm view'lar da silinir)
err = redis.InvalidateItem(ctx, "blog", blog.ID)
```

### Örnek 4: Parent Entity Güncellemesi

```go
// Author güncellendi
err := repo.UpdateAuthor(ctx, author)
if err != nil {
    return err
}

// Author'ın kendisi + ona bağlı tüm bloglar temizlenir
err = redis.InvalidateEntity(ctx, "author", author.ID)
```

### Örnek 5: Cache Warm-up (Önceden Yükleme)

```go
// DB'ye yazdıktan sonra cache'i hemen ısıt
blog, err := repo.CreateBlog(ctx, input)
if err != nil {
    return err
}

// Card view'ı hemen cache'le (ilk istek hızlı olsun)
err = redis.UpsertItem(ctx, "blog", blog.ToCardView(), 30*time.Minute,
    redis.GetOptions{View: "card"})
```

---

## 8. Paket Dosya Yapısı

```
pkg/redis/
├── client.go        # Redis bağlantı yönetimi (Singleton)
├── keys.go          # Key builder fonksiyonları
├── cache.go         # GetItem, GetList, UpsertItem
└── invalidation.go  # InvalidateItem, InvalidateByDependency, InvalidateEntity
```

---

## 9. Checklist: Yeni Entity Eklerken

```
□ Cacheable interface'i implemente et
  □ GetID() string
  □ GetDependencies() []Dependency

□ Key domain adını belirle (örn: "product", "order")

□ View tipleri belirle (örn: "card", "detail", "admin")

□ Service katmanında:
  □ Read fonksiyonlarında GetItem/GetList kullan
  □ Write fonksiyonlarından sonra InvalidateItem çağır
  □ Parent entity güncellemelerinde InvalidateEntity kullan

□ Singleflight ekle (yüksek trafikli endpoint'ler için)
```
