package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/okanay/go-template/configs"
	"github.com/okanay/go-template/pkg/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[MAIN::ENV] :: .env dosyası bulunamadı, sistem ortam değişkenleri kullanılacak.")
	}

	db, err := database.NewPostgres(os.Getenv("MAIN_CONN_STRING"))
	if err != nil {
		log.Fatalf("[MAIN::ERROR] :: Veritabanına bağlanılamadı: %v", err)
	}

	defer db.Close()
	log.Println("[MAIN::SUCCESS] :: Veritabanına başarıyla bağlanıldı.")

	router := gin.Default()
	router.Use(configs.CorsConfig())
	router.Use(configs.SecureConfig)
	router.SetTrustedProxies([]string{"192.168.1.2"})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Go Template API çalışıyor!",
			"env":     os.Getenv("MAIN_CONN_STRING"),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("[MAIN::INFO] :: PORT ortam değişkeni ayarlanmamış, varsayılan olarak 8080 kullanılıyor.")
	}

	serverAddr := fmt.Sprintf(":%s", port)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("[MAIN::ERROR] :: Sunucu başlatılamadı: %v", err)
	}
}
