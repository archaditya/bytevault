package server

import (
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
)

// registerUploadInviteRoutes configures endpoints for the Client File Portal (Guest Upload Invites).
func (s *Server) registerUploadInviteRoutes(v1 *echo.Group, protected *echo.Group, h *handler.UploadInviteHandler, userRepo *repository.UserRepository) {
	restrictCheck := appMiddleware.RestrictionCheck(userRepo)

	// Public guest endpoints (no auth)
	pub := v1.Group("/public/upload-invite")
	pub.GET("/:token", h.GetInviteInfo)
	pub.POST("/:token/verify-passcode", h.VerifyPasscode)
	pub.POST("/:token/upload-session", h.GuestCreateUploadSession)
	pub.POST("/:token/complete/:fileId", h.GuestCompleteUpload)

	// Protected owner endpoints (JWT required)
	invites := protected.Group("/upload-invites")
	invites.POST("", h.CreateInvite, restrictCheck)
	invites.GET("", h.ListInvites)
	invites.DELETE("/:id", h.RevokeInvite)
}
