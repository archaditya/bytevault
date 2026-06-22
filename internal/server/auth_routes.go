package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	"github.com/archaditya/bytevault/internal/service"
)

// registerAuthRoutes adds all /auth/* endpoints.
// These are public — register, login, refresh, logout don't need a token.
func (s *Server) registerAuthRoutes(v1 *Group, authService *service.AuthService) {
	authHandler := handler.NewAuthHandler(authService)

	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
}
