package server

import (
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
)

func (s *Server) registerShareRoutes(protected *echo.Group, sh *handler.ShareHandler, userRepo *repository.UserRepository) {
	restrictCheck := appMiddleware.RestrictionCheck(userRepo)
	shares := protected.Group("/shares")
	shares.POST("", sh.GrantShare, restrictCheck)
	shares.GET("", sh.ListResourceShares)
	shares.GET("/shared-with-me", sh.ListSharedWithMe)
	shares.DELETE("/:id", sh.RevokeShare)
}
