package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
)

// registerAuthRoutes adds all /auth/* endpoints.
func (s *Server) registerAuthRoutes(v1 *Group, protected *Group, authService *service.AuthService, notifHandler *handler.NotificationHandler, userRepo *repository.UserRepository) {
	authHandler := handler.NewAuthHandler(authService, userRepo)

	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/google", authHandler.GoogleLogin)

	auth.POST("/verify-email", notifHandler.VerifyEmail)
	auth.POST("/resend-otp", notifHandler.ResendOTP)
	auth.POST("/forgot-password", notifHandler.ForgotPassword)
	auth.POST("/reset-password", notifHandler.ResetPassword)

	// MFA Public route (verify-login doesn't need JWT, uses short-lived mfa_token)
	auth.POST("/mfa/verify-login", authHandler.MFAVerifyLogin)

	// MFA Protected routes (require active JWT session)
	mfa := protected.Group("/auth/mfa")
	mfa.POST("/setup", authHandler.MFASetup)
	mfa.POST("/enable", authHandler.MFAEnable)
	mfa.POST("/disable", authHandler.MFADisable)
}

// registerNotificationRoutes adds all protected notification endpoints.
func (s *Server) registerNotificationRoutes(protected *Group, notifHandler *handler.NotificationHandler) {
	protected.GET("/notifications", notifHandler.ListNotifications)
	protected.POST("/notifications/:id/read", notifHandler.MarkAsRead)
	protected.POST("/notifications/read-all", notifHandler.MarkAllAsRead)
	protected.POST("/push-tokens", notifHandler.RegisterPushToken)

	// Admin Broadcast & Logs
	protected.POST("/notifications/admin/send", notifHandler.SendAdminNotification, appMiddleware.RequirePermission("admin:users"))
	protected.GET("/admin/notifications", notifHandler.ListAllNotifications, appMiddleware.RequirePermission("admin:users"))
}
