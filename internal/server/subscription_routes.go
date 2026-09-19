package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	"github.com/labstack/echo/v4"
)

// registerSubscriptionRoutes wires user subscription, package, and webhook routes.
func (s *Server) registerSubscriptionRoutes(
	public *echo.Group,
	protected *echo.Group,
	subHandler *handler.SubscriptionHandler,
	pkgHandler *handler.PackageHandler,
	webhookHandler *handler.WebhookHandler,
) {
	// Webhooks (public endpoint, verified via Razorpay HMAC signature)
	public.POST("/webhooks/razorpay", webhookHandler.HandleRazorpayWebhook)

	// Public package discovery
	public.GET("/packages", pkgHandler.ListPackages)
	public.GET("/packages/:id", pkgHandler.GetPackage)

	// Public invoice view and download (unauthenticated, identified by secure random txn UUID)
	public.GET("/invoices/:id", subHandler.GetInvoiceHTMLPublic)
	public.GET("/invoices/:id/download", subHandler.DownloadInvoicePublic)

	// Protected user subscription management
	subGroup := protected.Group("/subscription")
	{
		subGroup.POST("/subscribe", subHandler.Subscribe)
		subGroup.POST("/verify", subHandler.Verify)
		subGroup.GET("/current", subHandler.GetCurrent)
		subGroup.POST("/upgrade", subHandler.Upgrade)
		subGroup.POST("/downgrade", subHandler.Downgrade)
		subGroup.POST("/cancel", subHandler.Cancel)
		subGroup.POST("/cancel-downgrade", subHandler.CancelPendingDowngrade)
		subGroup.GET("/transactions", subHandler.ListTransactions)
		subGroup.GET("/invoices/:id", subHandler.GetInvoiceHTML)
		subGroup.GET("/invoices/:id/download", subHandler.DownloadInvoice)
	}
}
