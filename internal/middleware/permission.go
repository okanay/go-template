package middleware

import (
	"github.com/gin-gonic/gin"
)

func (m *Manager) RequirePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
