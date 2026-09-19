package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/monitoring"
	"github.com/archaditya/bytevault/internal/razorpay"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// RazorpayWebhookPayload matches the structure sent by Razorpay webhook notifications.
type RazorpayWebhookPayload struct {
	ID        string   `json:"id"` // Unique event ID for idempotency
	Entity    string   `json:"entity"`
	AccountID string   `json:"account_id"`
	Event     string   `json:"event"`
	Contains  []string `json:"contains"`
	Payload   struct {
		Subscription struct {
			Entity struct {
				ID           string            `json:"id"`
				PlanID       string            `json:"plan_id"`
				CustomerID   string            `json:"customer_id"`
				Status       string            `json:"status"`
				CurrentStart int64             `json:"current_start"`
				CurrentEnd   int64             `json:"current_end"`
				Notes        map[string]string `json:"notes"`
			} `json:"entity"`
		} `json:"subscription"`
		Payment struct {
			Entity struct {
				ID       string `json:"id"`
				Amount   int    `json:"amount"`
				Currency string `json:"currency"`
				Status   string `json:"status"`
				OrderID  string `json:"order_id"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
	CreatedAt int64 `json:"created_at"`
}

type WebhookHandler struct {
	razorpayClient   *razorpay.Client
	subRepo          *repository.SubscriptionRepository
	pkgRepo          *repository.PackageRepository
	userRepo         *repository.UserRepository
	auditRepo        *repository.SubscriptionAuditRepository
	txnService       *service.TransactionService
	fileLogger       *monitoring.FileAuditLogger
	notifService     *service.NotificationService
	webhookEventRepo *repository.WebhookEventRepository
}

func NewWebhookHandler(
	rzpClient *razorpay.Client,
	subRepo *repository.SubscriptionRepository,
	pkgRepo *repository.PackageRepository,
	userRepo *repository.UserRepository,
	auditRepo *repository.SubscriptionAuditRepository,
	txnService *service.TransactionService,
	fileLogger *monitoring.FileAuditLogger,
	notifService *service.NotificationService,
	webhookEventRepo *repository.WebhookEventRepository,
) *WebhookHandler {
	return &WebhookHandler{
		razorpayClient:   rzpClient,
		subRepo:          subRepo,
		pkgRepo:          pkgRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
		txnService:       txnService,
		fileLogger:       fileLogger,
		notifService:     notifService,
		webhookEventRepo: webhookEventRepo,
	}
}

// HandleRazorpayWebhook receives and dispatches Razorpay webhook events (POST /api/v1/webhooks/razorpay).
// FIX #3: Processes synchronously — returns non-200 on failure so Razorpay retries.
// FIX #4: Idempotency — deduplicates events via processed_webhook_events table.
func (h *WebhookHandler) HandleRazorpayWebhook(c echo.Context) error {
	signature := c.Request().Header.Get("X-Razorpay-Signature")
	if signature == "" {
		return c.String(http.StatusBadRequest, "missing signature")
	}

	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "cannot read body")
	}

	// Verify webhook signature
	if h.razorpayClient != nil && !h.razorpayClient.VerifyWebhookSignature(bodyBytes, signature) {
		log.Warn().Str("signature", signature).Msg("Invalid Razorpay webhook signature")
		return c.String(http.StatusBadRequest, "invalid signature")
	}

	var event RazorpayWebhookPayload
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal Razorpay webhook JSON")
		return c.String(http.StatusBadRequest, "invalid json payload")
	}

	eventID := h.getEventID(event)
	log.Info().Str("event", event.Event).Str("event_id", eventID).
		Str("sub_id", event.Payload.Subscription.Entity.ID).Msg("Received Razorpay webhook")

	// Idempotency check: skip if already processed
	if h.webhookEventRepo != nil {
		exists, checkErr := h.webhookEventRepo.Exists(c.Request().Context(), eventID)
		if checkErr == nil && exists {
			log.Info().Str("event_id", eventID).Msg("Duplicate webhook event, returning OK")
			return c.JSON(http.StatusOK, map[string]string{"status": "already_processed"})
		}
	}

	// Process synchronously — Razorpay retries on non-200 response
	if err := h.processEvent(c.Request().Context(), event); err != nil {
		log.Error().Err(err).Str("event", event.Event).Str("event_id", eventID).
			Msg("Webhook processing failed — Razorpay will retry")
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "processing_failed"})
	}

	// Record processed event for future idempotency
	if h.webhookEventRepo != nil {
		if recErr := h.webhookEventRepo.Record(c.Request().Context(), eventID, event.Event, event.Payload.Subscription.Entity.ID); recErr != nil {
			log.Warn().Err(recErr).Str("event_id", eventID).Msg("Failed to record webhook event for idempotency")
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// getEventID returns a unique identifier for this webhook event for idempotency.
func (h *WebhookHandler) getEventID(ev RazorpayWebhookPayload) string {
	if ev.ID != "" {
		return ev.ID
	}
	// Fallback: generate composite key from event fields
	return fmt.Sprintf("%s_%s_%s_%d", ev.Event, ev.Payload.Subscription.Entity.ID, ev.Payload.Payment.Entity.ID, ev.CreatedAt)
}

// freeTierLimits returns free tier storage limits from DB, with hardcoded fallback.
func (h *WebhookHandler) freeTierLimits(ctx context.Context) (int64, int64) {
	freePkg, err := h.pkgRepo.FindByName(ctx, "free")
	if err == nil && freePkg != nil {
		return freePkg.StorageLimitBytes, freePkg.MaxFileSizeBytes
	}
	return model.DefaultFreeStorageLimitBytes, model.DefaultFreeMaxFileSizeBytes
}

// processEvent dispatches a webhook event to the appropriate handler.
// Returns an error on critical failures (DB writes) so the caller can return non-200 for retry.
// Non-critical failures (notifications) are logged but don't cause a retry.
func (h *WebhookHandler) processEvent(ctx context.Context, ev RazorpayWebhookPayload) error {
	subID := ev.Payload.Subscription.Entity.ID
	if subID == "" {
		return nil
	}

	sub, err := h.subRepo.FindByRazorpayID(ctx, subID)
	if err != nil || sub == nil {
		log.Warn().Str("sub_id", subID).Msg("Subscription not found for webhook event")
		return nil // Don't retry — subscription doesn't exist in our DB
	}

	var periodStart, periodEnd *time.Time
	if ev.Payload.Subscription.Entity.CurrentStart > 0 {
		st := time.Unix(ev.Payload.Subscription.Entity.CurrentStart, 0).UTC()
		periodStart = &st
	}
	if ev.Payload.Subscription.Entity.CurrentEnd > 0 {
		en := time.Unix(ev.Payload.Subscription.Entity.CurrentEnd, 0).UTC()
		periodEnd = &en
	}

	paymentID := ev.Payload.Payment.Entity.ID
	var pID *string
	if paymentID != "" {
		pID = &paymentID
	}

	switch ev.Event {
	case "subscription.authenticated":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusAuthenticated, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update authenticated status: %w", err)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.activated":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusActive, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update activated status: %w", err)
		}
		pkg, pkgErr := h.pkgRepo.FindByID(ctx, sub.PackageID)
		if pkgErr == nil && pkg != nil {
			if limitErr := h.userRepo.UpdateStorageLimits(ctx, sub.UserID, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes); limitErr != nil {
				log.Error().Err(limitErr).Str("user_id", sub.UserID).Msg("Failed to update storage on activation")
			}
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.charged":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusActive, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update charged status: %w", err)
		}

		// New billing cycle charged — unlock plan change restrictions for the new cycle
		_ = h.subRepo.SetUpgradedFrom(ctx, sub.ID, nil)

		// Reset renewal reminder flags for the new cycle
		if sub.Metadata != nil {
			for _, d := range []int{7, 5, 3, 2, 1} {
				delete(sub.Metadata, fmt.Sprintf("reminder_%dd_sent", d))
			}
			_ = h.subRepo.UpdateMetadata(ctx, sub.ID, sub.Metadata)
		}

		if paymentID != "" {
			totalPaise := ev.Payload.Payment.Entity.Amount
			currency := ev.Payload.Payment.Entity.Currency
			orderID := ev.Payload.Payment.Entity.OrderID
			txn, txnErr := h.txnService.RecordCharge(ctx, sub, paymentID, orderID, totalPaise, currency)
			if txnErr != nil {
				h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", fmt.Sprintf("record charge: %s", txnErr.Error()))
				return fmt.Errorf("record charge: %w", txnErr)
			}

			// FIX #18: Single notification path — receipt email is sent by RecordCharge,
			// NotifySubscription handles in-app + push + branded email notification.
			if h.notifService != nil {
				totalINR := fmt.Sprintf("%.2f", float64(totalPaise)/100.0)
				invNum := "BV-INVOICE"
				if txn != nil && txn.InvoiceNumber != nil {
					invNum = *txn.InvoiceNumber
				}
				_ = h.notifService.NotifySubscription(
					ctx, sub.UserID,
					"Payment Successful",
					fmt.Sprintf("Your recurring payment of ₹%s for ByteVault was charged successfully. Invoice: %s.", totalINR, invNum),
					"subscription.charged",
				)
			}
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	// FIX #20: Separate completed vs expired — different statuses and notifications
	case "subscription.completed":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusCompleted, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update completed status: %w", err)
		}
		storageLimit, maxFileSize := h.freeTierLimits(ctx)
		if limitErr := h.userRepo.UpdateStorageLimits(ctx, sub.UserID, storageLimit, maxFileSize); limitErr != nil {
			log.Error().Err(limitErr).Msg("Failed to reset storage on completion")
		}
		if h.notifService != nil {
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Subscription Completed",
				"Your paid subscription cycle has completed its full billing term. Your storage has been reset to the Free tier.",
				"subscription.completed",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.expired":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusExpired, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update expired status: %w", err)
		}
		storageLimit, maxFileSize := h.freeTierLimits(ctx)
		if limitErr := h.userRepo.UpdateStorageLimits(ctx, sub.UserID, storageLimit, maxFileSize); limitErr != nil {
			log.Error().Err(limitErr).Msg("Failed to reset storage on expiry")
		}
		if h.notifService != nil {
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Subscription Expired",
				"Your subscription has expired due to failed payment attempts. Your storage has been reset to the Free tier. Please resubscribe to restore access.",
				"subscription.expired",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	// FIX #8: Reset storage on cancellation if period already ended. Don't set cancel_at_cycle_end=true
	// since the cancellation has already been executed by Razorpay.
	case "subscription.cancelled":
		now := time.Now().UTC()
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusCancelled, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update cancelled status: %w", err)
		}
		// cancel_at_cycle_end = false: the cancellation has been executed (not just scheduled)
		if err := h.subRepo.SetCancellation(ctx, sub.ID, false, &now); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("set cancellation: %w", err)
		}

		// Reset storage if period has already ended or cancel was immediate
		if periodEnd == nil || !periodEnd.After(now) {
			storageLimit, maxFileSize := h.freeTierLimits(ctx)
			if limitErr := h.userRepo.UpdateStorageLimits(ctx, sub.UserID, storageLimit, maxFileSize); limitErr != nil {
				log.Error().Err(limitErr).Msg("Failed to reset storage on cancellation")
			}
		}
		// If period is still in the future, the scheduler will reset storage when it expires

		if h.notifService != nil {
			endStr := "immediately"
			if periodEnd != nil && periodEnd.After(now) {
				endStr = "on " + periodEnd.Format("02 Jan 2006")
			}
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Subscription Cancelled",
				fmt.Sprintf("Your ByteVault subscription has been cancelled. Access expires %s.", endStr),
				"subscription.cancelled",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.halted":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusHalted, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update halted status: %w", err)
		}
		if h.notifService != nil {
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Subscription Halted — Action Required",
				"All automatic payment retry attempts have failed and your subscription has been halted. Please update your payment method in Settings to avoid downgrade to Free tier.",
				"subscription.halted",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.pending":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusPending, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update pending status: %w", err)
		}
		if h.notifService != nil {
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Payment Failed — Retry Scheduled",
				"We were unable to process your recurring subscription charge. Razorpay will automatically retry the charge shortly. Please ensure your card or bank account has sufficient funds.",
				"subscription.pending",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.paused":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusPaused, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update paused status: %w", err)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.resumed":
		if err := h.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusActive, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update resumed status: %w", err)
		}
		if h.notifService != nil {
			_ = h.notifService.NotifySubscription(
				ctx, sub.UserID,
				"Subscription Resumed",
				"Your ByteVault subscription has been successfully resumed.",
				"subscription.resumed",
			)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	case "subscription.updated":
		// Razorpay updated subscription plan (e.g. scheduled upgrade/downgrade took effect)
		newPlanID := ev.Payload.Subscription.Entity.PlanID
		status := ev.Payload.Subscription.Entity.Status
		if status == "" {
			status = model.SubscriptionStatusActive
		}

		if newPlanID != "" {
			newPkg, pkgErr := h.pkgRepo.FindByRazorpayPlanID(ctx, newPlanID)
			if pkgErr == nil && newPkg != nil {
				if err := h.subRepo.UpdatePackageID(ctx, sub.ID, newPkg.ID); err != nil {
					h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", fmt.Sprintf("update package: %s", err.Error()))
					return fmt.Errorf("update subscription package on updated event: %w", err)
				}

				// Clear pending downgrade since plan change was applied
				if sub.PendingPackageID != nil {
					_ = h.subRepo.SetPendingDowngrade(ctx, sub.ID, nil)
				}

				// Adjust storage limits to new plan
				if limitErr := h.userRepo.UpdateStorageLimits(ctx, sub.UserID, newPkg.StorageLimitBytes, newPkg.MaxFileSizeBytes); limitErr != nil {
					log.Error().Err(limitErr).Msg("Failed to update storage on subscription.updated")
				}

				if h.notifService != nil {
					_ = h.notifService.NotifySubscription(
						ctx, sub.UserID,
						"Subscription Plan Updated",
						fmt.Sprintf("Your ByteVault subscription has been updated to %s.", newPkg.DisplayName),
						"subscription.updated",
					)
				}
			} else {
				log.Warn().Str("plan_id", newPlanID).Msg("Received subscription.updated with unknown plan ID")
			}
		}

		if err := h.subRepo.UpdateStatus(ctx, sub.ID, status, periodStart, periodEnd, pID); err != nil {
			h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "error", err.Error())
			return fmt.Errorf("update status on subscription.updated: %w", err)
		}
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "success", "")

	default:
		h.logWebhook(ctx, sub.UserID, sub.ID, ev.Event, "skipped", "unhandled event")
	}

	return nil
}

// logWebhook writes webhook event to both DB audit log and file audit log.
// FIX #19: Logs warnings on write failures instead of silently discarding.
func (h *WebhookHandler) logWebhook(ctx context.Context, userID, subID, eventType, status, errMsg string) {
	entry := &model.SubscriptionAuditLog{
		UserID:         &userID,
		SubscriptionID: &subID,
		EventType:      "webhook." + eventType,
		EventSource:    "webhook",
		Status:         status,
	}
	if errMsg != "" {
		entry.ErrorMessage = &errMsg
	}
	if dbErr := h.auditRepo.Log(ctx, entry); dbErr != nil {
		log.Warn().Err(dbErr).Str("event_type", eventType).Msg("Failed to write webhook audit log to database")
	}

	if h.fileLogger != nil {
		if fileErr := h.fileLogger.Log(monitoring.AuditEvent{
			EventType:      "webhook." + eventType,
			EventSource:    "webhook",
			UserID:         userID,
			SubscriptionID: subID,
			Status:         status,
			ErrorMessage:   errMsg,
		}); fileErr != nil {
			log.Warn().Err(fileErr).Str("event_type", eventType).Msg("Failed to write webhook audit log to file")
		}
	}
}
