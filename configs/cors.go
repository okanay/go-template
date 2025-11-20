package configs

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsConfig() gin.HandlerFunc {
	var origins = []string{
		// Production domains
		"https://mydomain.com",
		"https://www.mydomain.com",

		// Local development
		"https://local.mydomain.com",
		"https://www.local.mydomain.com",
	}

	// Debug mode'da localhost ekle
	if gin.Mode() == gin.DebugMode {
		origins = append(origins,
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		)
	}

	return cors.New(cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{
			"GET",
			"PUT",
			"POST",
			"DELETE",
			"HEAD",
			"OPTIONS",
			"PATCH",
		},
		AllowHeaders: []string{
			"Content-Type",
			"Authorization",
			"Accept",
			"Origin",
			"X-Requested-With",
			"Cache-Control",
			"X-Language",
			"X-Currency",
			"X-CSRF-Token",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"X-Request-Id",
		},
		AllowCredentials: true,
		MaxAge:           60 * 24 * 30,
	})
}
