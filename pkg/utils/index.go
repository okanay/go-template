package utils

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// GenerateRandomString: Belirtilen uzunlukta güvenli rastgele string üretir.
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(Alphabet))))
		if err != nil {
			// crypto/rand hatası kritik bir OS sorunudur, panic makuldür.
			log.Panicf("crypto/rand failed: %v", err)
		}
		b[i] = Alphabet[num.Int64()]
	}
	return string(b)
}

// GenerateRandomInt: [min, max] aralığında güvenli rastgele sayı üretir.
func GenerateRandomInt(min, max int) int {
	if max <= min {
		return min // Hata yerine min dönerek akışı bozmuyoruz, log basılabilir.
	}
	diff := max - min
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		log.Panicf("crypto/rand failed: %v", err)
	}
	return int(nBig.Int64()) + min
}

// SanitizeFilename: Dosya adını güvenli hale getirir.
func SanitizeFilename(filename string) string {
	// 1. Uzantıyı ayır (varsa korumak için) ama biz tamamını sanitize edeceğiz.
	// 2. Boşlukları tire yap
	filename = strings.ReplaceAll(filename, " ", "-")

	// 3. Sadece izin verilen karakterleri tut
	filename = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return -1 // Geçersiz karakteri sil
	}, filename)

	// 4. Çift noktaları (..) engelle (Path traversal koruması)
	return strings.ReplaceAll(filename, "..", "")
}

// Slugify: URL dostu string oluşturur (Türkçe karakter desteği eklenebilir ama basic tutuldu).
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Regex ile alfanumerik olmayan her şeyi tire yap
	// Bu yöntem senin döngüden daha temizdir ve ardışık tireleri engeller.
	reg, _ := regexp.Compile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")

	// Baştaki ve sondaki tireleri temizle
	return strings.Trim(s, "-")
}

// CollapseSpaces: Birden fazla boşluğu tek boşluğa indirir.
func CollapseSpaces(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

// Now: Her zaman UTC zaman döner.
func Now() time.Time {
	return time.Now().UTC()
}

// FormatDate: Zamanı "DD.MM.YYYY" formatında döner.
func FormatDate(t time.Time) string {
	return t.Format("02.01.2006")
}

// FormatDateTime: Zamanı "DD.MM.YYYY HH:mm" formatında döner.
func FormatDateTime(t time.Time) string {
	return t.Format("02.01.2006 15:04")
}

// GetEnv: Ortam değişkenini okur, yoksa default değeri döner.
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvInt: Ortam değişkenini int olarak okur, yoksa default döner.
func GetEnvInt(key string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetEnvBool: Ortam değişkenini bool olarak okur (true, 1, yes), yoksa default döner.
func GetEnvBool(key string, defaultValue bool) bool {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// ToJSON: Struct'ı string JSON'a çevirir (Loglama ve Debug için çok yararlı).
func ToJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// FromJSON: JSON string'ini belirtilen Generic tipe çevirir ve pointer döner.
// Kullanım: user, err := utils.FromJSON[User](jsonString)
func FromJSON[T any](jsonStr string) (*T, error) {
	var target T
	if err := json.Unmarshal([]byte(jsonStr), &target); err != nil {
		return nil, err
	}
	return &target, nil
}

// MapToStruct: map[string]any tipindeki veriyi struct'a çevirir.
// Genelde JSON'dan map'e dönmüş veriyi struct'a maplemek için kullanılır.
func MapToStruct[T any](data map[string]any) (*T, error) {
	// En güvenli yol: Map -> JSON -> Struct
	// Performans kritiği varsa 'mitchellh/mapstructure' paketi kullanılabilir ama bu native yoldur.
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var target T
	if err := json.Unmarshal(bytes, &target); err != nil {
		return nil, err
	}
	return &target, nil
}
