package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/okanay/go-template/internal/auth"
)

func (m *Manager) RequirePermission(perm auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
