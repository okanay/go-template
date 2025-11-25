package middleware

import "github.com/okanay/go-template/internal/auth"

type Manager struct {
	authService *auth.Service
}

func NewManager(authService *auth.Service) *Manager {
	return &Manager{
		authService: authService,
	}
}
