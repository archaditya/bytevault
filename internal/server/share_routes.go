package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

func (s *Server) registerShareRoutes(protected *echo.Group, sh *handler.ShareHandler) {
	shares := protected.Group("/shares")
	shares.POST("", sh.GrantShare)
	shares.GET("", sh.ListResourceShares)
	shares.GET("/shared-with-me", sh.ListSharedWithMe)
	shares.DELETE("/:id", sh.RevokeShare)
}
