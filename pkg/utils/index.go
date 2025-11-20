package utils

import (
	"crypto/rand"
	"log"
	"math/big"
	"strings"
	"time"
)

const Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Güvenli rastgele string üretimi (CSPRNG ile)
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(Alphabet))))
		if err != nil {
			log.Fatalf("crypto/rand failed: %v", err)
		}
		b[i] = Alphabet[num.Int64()]
	}
	return string(b)
}

// Güvenli rastgele sayı üretimi (CSPRNG ile)
func GenerateRandomInt(min, max int) int {
	if max <= min {
		log.Fatalf("Invalid range: max (%d) must be greater than min (%d)", max, min)
	}
	diff := max - min
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		log.Fatalf("crypto/rand failed: %v", err)
	}
	return int(nBig.Int64()) + min
}

// Dosya adını güvenli hale getiren yardımcı fonksiyon
func SanitizeFilename(filename string) string {
	// Boşlukları tire ile değiştir
	sanitized := strings.ReplaceAll(filename, " ", "-")

	// Sadece alfanumerik, nokta, tire ve alt çizgi karakterlerine izin ver
	sanitized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, sanitized)

	return sanitized
}

// Zaman ölçümü için kullanılır
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	if elapsed <= 5*time.Millisecond {
		return
	}
	log.Printf("%s ~TOOK~ %s", name, elapsed.Round(time.Millisecond))
}

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove all characters that are not letters, numbers, or hyphens
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('0' <= b && b <= '9') ||
			b == '-' {
			result.WriteByte(b)
		}
	}
	return result.String()
}
