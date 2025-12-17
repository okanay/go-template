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
	if err := godotenv.Load(); err != nil {
		log.Println("[MAIN::ENV] :: .env file not found, system environment variables will be used.")
	}

	db, err := database.NewPostgres(os.Getenv("DB_MAIN_CONN_STRING"))
	if err != nil {
		log.Fatalf("[MAIN::ERROR] :: Failed to connect to database: %v", err)
	}

	defer db.Close()
	log.Println("[MAIN::SUCCESS] :: Successfully connected to the database.")

	redisAddr := os.Getenv("REDIS_ADDR")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPass := os.Getenv("REDIS_PASS")
	redisDB := os.Getenv("REDIS_DB")

	_, err = redis.NewRedisClient(
		[]string{redisAddr},
		redisUsername,
		redisPass,
		redisDB,
	)

	if err != nil {
		log.Fatalf("[MAIN::ERROR] :: Redis connection failed: %v", err)
	}

	log.Println("[MAIN::SUCCESS] :: Redis connection successful.")

	router := gin.Default()
	router.Use(configs.CorsConfig())
	router.Use(configs.SecureConfig)
	router.SetTrustedProxies([]string{"192.168.1.2"})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Go Template API is running!",
			"env":     os.Getenv("MAIN_CONN_STRING"),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("[MAIN::INFO] :: PORT environment variable not set, using default 8080.")
	}

	serverAddr := fmt.Sprintf(":%s", port)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("[MAIN::ERROR] :: Failed to start server: %v", err)
	}
}
