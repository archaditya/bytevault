package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
)

// registerAuthRoutes adds all /auth/* endpoints.
// These are public — register, login, refresh, logout, verify-email, resend-otp don't need a token.
func (s *Server) registerAuthRoutes(v1 *Group, authService *service.AuthService, notifHandler *handler.NotificationHandler, userRepo *repository.UserRepository) {
	authHandler := handler.NewAuthHandler(authService, userRepo)

	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/google", authHandler.GoogleLogin)


	// OTP verification (public — user isn't authenticated yet)
	auth.POST("/verify-email", notifHandler.VerifyEmail)
	auth.POST("/resend-otp", notifHandler.ResendOTP)
	auth.POST("/forgot-password", notifHandler.ForgotPassword)
	auth.POST("/reset-password", notifHandler.ResetPassword)
}

// registerNotificationRoutes adds all protected notification endpoints.
func (s *Server) registerNotificationRoutes(protected *Group, notifHandler *handler.NotificationHandler) {
	protected.GET("/notifications", notifHandler.ListNotifications)
	protected.POST("/notifications/:id/read", notifHandler.MarkAsRead)
	protected.POST("/notifications/read-all", notifHandler.MarkAllAsRead)
	protected.POST("/push-tokens", notifHandler.RegisterPushToken)
}
