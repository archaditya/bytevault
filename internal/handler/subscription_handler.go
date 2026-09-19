package handler

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
)

type SubscriptionHandler struct {
	subService    *service.SubscriptionService
	txnService    *service.TransactionService
	razorpayKeyID string
}

func NewSubscriptionHandler(
	subService *service.SubscriptionService,
	txnService *service.TransactionService,
	razorpayKeyID string,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subService:    subService,
		txnService:    txnService,
		razorpayKeyID: razorpayKeyID,
	}
}

// Subscribe initiates a recurring subscription with Razorpay (POST /api/v1/subscription/subscribe).
func (h *SubscriptionHandler) Subscribe(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		PackageID string `json:"package_id"`
	}
	if err := c.Bind(&req); err != nil || req.PackageID == "" {
		return SendError(c, http.StatusBadRequest, "package_id is required")
	}

	sub, err := h.subService.Subscribe(c.Request().Context(), userID, req.PackageID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	rzpSubID := ""
	if sub.RazorpaySubscriptionID != nil {
		rzpSubID = *sub.RazorpaySubscriptionID
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"subscription_id":          sub.ID,
		"razorpay_subscription_id": rzpSubID,
		"razorpay_key_id":          h.razorpayKeyID,
		"package":                  sub.Package,
	}, nil)
}

// Verify validates payment signature from Razorpay Checkout modal (POST /api/v1/subscription/verify).
func (h *SubscriptionHandler) Verify(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		RazorpaySubscriptionID string `json:"razorpay_subscription_id"`
		RazorpayPaymentID      string `json:"razorpay_payment_id"`
		RazorpaySignature      string `json:"razorpay_signature"`
	}
	if err := c.Bind(&req); err != nil || req.RazorpaySubscriptionID == "" || req.RazorpayPaymentID == "" || req.RazorpaySignature == "" {
		return SendError(c, http.StatusBadRequest, "razorpay_subscription_id, razorpay_payment_id, and razorpay_signature are required")
	}

	sub, err := h.subService.Verify(c.Request().Context(), userID, req.RazorpaySubscriptionID, req.RazorpayPaymentID, req.RazorpaySignature)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	// Record initial transaction and dispatch Brevo invoice email immediately
	if h.txnService != nil && sub != nil && sub.Package != nil && sub.Package.PricePaise > 0 {
		totalPaise := sub.Package.PriceWithGSTPaise
		if totalPaise <= 0 {
			totalPaise = sub.Package.PricePaise + int(math.Round(float64(sub.Package.PricePaise)*0.18))
		}
		go func(s *model.Subscription, pID string, paise int) {
			bgCtx := context.Background()
			_, _ = h.txnService.RecordCharge(bgCtx, s, pID, "", paise, "INR")
		}(sub, req.RazorpayPaymentID, totalPaise)
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// GetCurrent retrieves the active subscription for the current user (GET /api/v1/subscription/current).
func (h *SubscriptionHandler) GetCurrent(c echo.Context) error {
	userID := c.Get("user_id").(string)

	sub, err := h.subService.GetCurrentSubscription(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// Upgrade upgrades the current plan immediately (POST /api/v1/subscription/upgrade).
func (h *SubscriptionHandler) Upgrade(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		PackageID string `json:"package_id"`
	}
	if err := c.Bind(&req); err != nil || req.PackageID == "" {
		return SendError(c, http.StatusBadRequest, "package_id is required")
	}

	sub, err := h.subService.Upgrade(c.Request().Context(), userID, req.PackageID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// Downgrade schedules a downgrade at billing cycle end (POST /api/v1/subscription/downgrade).
func (h *SubscriptionHandler) Downgrade(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		PackageID string `json:"package_id"`
	}
	if err := c.Bind(&req); err != nil || req.PackageID == "" {
		return SendError(c, http.StatusBadRequest, "package_id is required")
	}

	sub, err := h.subService.Downgrade(c.Request().Context(), userID, req.PackageID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// Cancel requests cancellation at billing cycle end (POST /api/v1/subscription/cancel).
func (h *SubscriptionHandler) Cancel(c echo.Context) error {
	userID := c.Get("user_id").(string)

	sub, err := h.subService.Cancel(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// CancelPendingDowngrade revokes a scheduled plan downgrade (POST /api/v1/subscription/cancel-downgrade).
func (h *SubscriptionHandler) CancelPendingDowngrade(c echo.Context) error {
	userID := c.Get("user_id").(string)

	sub, err := h.subService.CancelPendingDowngrade(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// ListTransactions retrieves the authenticated user's transactions (GET /api/v1/subscription/transactions).
func (h *SubscriptionHandler) ListTransactions(c echo.Context) error {
	userID := c.Get("user_id").(string)

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	txns, total, err := h.txnService.ListUserTransactions(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, txns, PaginationMetadata{
		Total: total,
		Limit: limit,
	})
}

// GetInvoiceHTML renders the HTML invoice for browser preview (GET /api/v1/subscription/invoices/:id).
// FIX #12: Validates that the requesting user owns the transaction, or is an admin.
func (h *SubscriptionHandler) GetInvoiceHTML(c echo.Context) error {
	userID := c.Get("user_id").(string)
	role, _ := c.Get("role").(string)
	txnID := c.Param("id")

	txn, err := h.txnService.GetTransactionByID(c.Request().Context(), txnID)
	if err != nil || txn == nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	if txn.UserID != userID && role != "admin" && role != "superadmin" {
		return SendError(c, http.StatusForbidden, "unauthorized to view this invoice")
	}

	html, err := h.txnService.GetInvoiceHTML(c.Request().Context(), txnID)
	if err != nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	return c.HTML(http.StatusOK, html)
}

// DownloadInvoice serves the HTML invoice with download disposition (GET /api/v1/subscription/invoices/:id/download).
// FIX #12: Validates that the requesting user owns the transaction, or is an admin.
func (h *SubscriptionHandler) DownloadInvoice(c echo.Context) error {
	userID := c.Get("user_id").(string)
	role, _ := c.Get("role").(string)
	txnID := c.Param("id")

	txn, err := h.txnService.GetTransactionByID(c.Request().Context(), txnID)
	if err != nil || txn == nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	if txn.UserID != userID && role != "admin" && role != "superadmin" {
		return SendError(c, http.StatusForbidden, "unauthorized to download this invoice")
	}

	html, err := h.txnService.GetInvoiceHTML(c.Request().Context(), txnID)
	if err != nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=invoice-%s.html", txnID))
	return c.HTML(http.StatusOK, html)
}

// GetInvoiceHTMLPublic serves the HTML invoice for browser preview in new tabs or from email links (GET /api/v1/invoices/:id).
func (h *SubscriptionHandler) GetInvoiceHTMLPublic(c echo.Context) error {
	txnID := c.Param("id")
	txn, err := h.txnService.GetTransactionByID(c.Request().Context(), txnID)
	if err != nil || txn == nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	html, err := h.txnService.GetInvoiceHTML(c.Request().Context(), txnID)
	if err != nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	return c.HTML(http.StatusOK, html)
}

// DownloadInvoicePublic serves the HTML invoice with download disposition (GET /api/v1/invoices/:id/download).
func (h *SubscriptionHandler) DownloadInvoicePublic(c echo.Context) error {
	txnID := c.Param("id")
	txn, err := h.txnService.GetTransactionByID(c.Request().Context(), txnID)
	if err != nil || txn == nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	html, err := h.txnService.GetInvoiceHTML(c.Request().Context(), txnID)
	if err != nil {
		return SendError(c, http.StatusNotFound, "invoice not found")
	}

	filename := fmt.Sprintf("PushPortVault-Invoice-%s.html", txnID)
	if txn.InvoiceNumber != nil && *txn.InvoiceNumber != "" {
		filename = fmt.Sprintf("PushPortVault-Invoice-%s.html", *txn.InvoiceNumber)
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.HTML(http.StatusOK, html)
}
