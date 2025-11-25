package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/okanay/go-template/internal/auth"
)

func (m *Manager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie(auth.AccessTokenCookieName)
		if err != nil {
			m.handleTokenRenewal(c)
			return
		}

		claims, err := auth.ValidateToken(accessToken)
		if err != nil {
			m.handleTokenRenewal(c)
			return
		}

		setContextValues(c, claims)
		c.Next()
	}
}

func (m *Manager) handleTokenRenewal(c *gin.Context) {
	// refreshToken, err := c.Cookie(auth.RefreshTokenCookieName)
	// if err != nil {
	// 	auth.ClearCookies(c)
	// 	apierror.Error(c, http.StatusUnauthorized, apierror.ErrUnauthorized, "Session expired, please login again")
	// 	return
	// }

	// newAccess, newRefresh, claims, err := m.authService.RefreshSession(c.Request.Context(), refreshToken)
	// if err != nil {
	// 	auth.ClearCookies(c)
	// 	apierror.Error(c, http.StatusUnauthorized, apierror.ErrUnauthorized, apierror.ErrorMessage("Invalid session: "+err.Error()))
	// 	return
	// }

	// auth.SetCookies(c, newAccess, newRefresh)

	// setContextValues(c, claims)
	c.Next()
}

func setContextValues(c *gin.Context, claims *auth.Claims) {
	c.Set("userID", claims.UserID)
	c.Set("role", claims.Role)
}
