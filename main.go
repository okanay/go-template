package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/okanay/go-template/configs"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[MAIN::ENV] :: .env dosyası bulunamadı, sistem ortam değişkenleri kullanılacak.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("[MAIN::INFO] :: PORT ortam değişkeni ayarlanmamış, varsayılan olarak 8080 kullanılıyor.")
	}

	router := gin.Default()
	router.Use(configs.CorsConfig())
	router.Use(configs.SecureConfig)

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hi World! 🚀",
			"status":  "working",
		})
	})

	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("[MAIN::SUCCESS] :: Sunucu http://localhost%s adresinde çalışıyor...", serverAddr)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("[MAIN::ERROR] :: Sunucu başlatılamadı: %v", err)
	}
}
