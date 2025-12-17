package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/okanay/go-template/configs"
	"github.com/okanay/go-template/pkg/database"
	"github.com/okanay/go-template/pkg/redis"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. ENVIRONMENT VARIABLES - .env dosyasını yükle
	// -------------------------------------------------------------------------
	// godotenv.Load() başarısız olursa sistem environment variable'ları kullanılır.
	// Bu durum production ortamında normaldir (Docker, Kubernetes vb.)
	if err := godotenv.Load(); err != nil {
		log.Println("[MAIN::ENV] :: .env file not found, system environment variables will be used.")
	}

	// -------------------------------------------------------------------------
	// 2. DATABASE CONNECTION - PostgreSQL bağlantısı
	// -------------------------------------------------------------------------
	// Veritabanı bağlantı string'i DB_MAIN_CONN_STRING env variable'ından alınır.
	// Bağlantı başarısız olursa uygulama tamamen durur (Fatalf).
	db, err := database.NewPostgres(os.Getenv("DB_MAIN_CONN_STRING"))
	if err != nil {
		log.Fatalf("[DB::ERROR] :: Failed to connect to database: %v", err)
	}

	// defer ile main fonksiyonu bittiğinde veritabanı bağlantısı kapatılır.
	// Bu, resource leak'i önlemek için kritik öneme sahiptir.
	defer db.Close()

	log.Println("[DB::SUCCESS] :: Successfully connected to the database.")

	// -------------------------------------------------------------------------
	// 3. REDIS CONNECTION - Redis bağlantısı ve konfigürasyonu
	// -------------------------------------------------------------------------
	// Redis bağlantı bilgileri environment variable'lardan okunur.
	redisAddr := os.Getenv("REDIS_ADDR")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPass := os.Getenv("REDIS_PASS")
	redisDB := os.Getenv("REDIS_DB")

	// Redis client'ı initialize edilir.
	// İlk parametre slice olarak geçiyor - cluster desteği için tasarlanmış.
	err = redis.Initialize(
		[]string{redisAddr},
		redisUsername,
		redisPass,
		redisDB,
	)

	// -------------------------------------------------------------------------
	// 3.1 DEV MODE - Geliştirme ortamında Redis cache temizliği
	// -------------------------------------------------------------------------
	// GIN_MODE=debug ise tüm Redis cache'i temizlenir.
	// Bu, development sırasında eski/stale cache verilerini önler.
	if os.Getenv("GIN_MODE") == "debug" {
		log.Println("[REDIS::INFO] :: DEV MODE: Clearing Redis cache...")

		if err := redis.InvalidateFlushDB(); err != nil {
			log.Printf("[REDIS::ERROR] :: Error clearing cache: %v", err)
		} else {
			log.Println("[REDIS::SUCCESS] :: DEV MODE: Redis cache cleared successfully.")
		}
	}

	// Redis bağlantı hatası kontrolü
	// NOT: Bu kontrol Initialize'dan sonra yapılıyor - yukarıdaki cache temizliği
	// bağlantı başarılı olsa bile çalışmayabilir.
	if err != nil {
		log.Fatalf("[REDIS::ERROR] :: Redis connection failed: %v", err)
	}

	log.Println("[REDIS::SUCCESS] :: Redis connection successful.")

	// -------------------------------------------------------------------------
	// 4. GIN ROUTER SETUP - HTTP Router konfigürasyonu
	// -------------------------------------------------------------------------
	// gin.Default() Logger ve Recovery middleware'lerini otomatik ekler.
	// - Logger: Her request'i loglar
	// - Recovery: Panic'leri yakalar ve 500 döner
	router := gin.Default()

	// -------------------------------------------------------------------------
	// 4.1 MIDDLEWARE CONFIGURATION
	// -------------------------------------------------------------------------
	// CORS middleware - Cross-Origin Resource Sharing ayarları
	// Frontend'in farklı bir domain'den API'ye erişmesine izin verir.
	router.Use(configs.CorsConfig())

	// Security middleware - Güvenlik header'larını ekler
	// (X-Content-Type-Options, X-Frame-Options, vb.)
	router.Use(configs.SecureConfig)

	// NOT :: IP takibi yaparken CF-Connecting-IP kullanacağız.
	// >>> Bu Nginx katmanında çalışan projeler için bir güvenlik katmanı.
	// X-Forwarded-For header'ı sadece bu IP'lerden geldiğinde güvenilir kabul edilir.
	// VPS'te Nginx ile sarmalanan bir projede LocalHost değerleri eklememiz gerekiyor.
	router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// -------------------------------------------------------------------------
	// 5. ROUTES - API endpoint tanımlamaları
	// -------------------------------------------------------------------------
	// Health check / root endpoint
	// API'nin çalışır durumda olduğunu doğrulamak için kullanılır.
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Go Template API is running!",
			"env":     os.Getenv("MAIN_CONN_STRING"),
		})
	})

	// -------------------------------------------------------------------------
	// 6. SERVER START - HTTP sunucusunu başlat
	// -------------------------------------------------------------------------
	// Port belirleme - önce environment variable, yoksa default 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("[MAIN::INFO] :: PORT environment variable not set, using default 8080.")
	}

	// Server adresi formatlanır (örn: ":8080")
	serverAddr := fmt.Sprintf(":%s", port)

	// Sunucu başlatılır - bu satır blocking'dir.
	// Hata durumunda uygulama tamamen durur.
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("[MAIN::ERROR] :: Failed to start server: %v", err)
	}
}
