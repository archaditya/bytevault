package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/monitoring"
	"github.com/archaditya/bytevault/internal/razorpay"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/rs/zerolog/log"
)

var (
	ErrActiveSubscriptionExists = errors.New("user already has an active subscription")
	ErrNoActiveSubscription     = errors.New("no active subscription found")
	ErrInvalidPlanTransition    = errors.New("invalid plan transition")
	ErrFreeTierNoPayment        = errors.New("free tier does not require payment")
	ErrSamePlan                 = errors.New("target plan is the same as current plan")
)

type SubscriptionService struct {
	subRepo        *repository.SubscriptionRepository
	pkgRepo        *repository.PackageRepository
	userRepo       *repository.UserRepository
	auditRepo      *repository.SubscriptionAuditRepository
	razorpayClient *razorpay.Client
	fileLogger     *monitoring.FileAuditLogger
	notifService   *NotificationService
}

func NewSubscriptionService(
	subRepo *repository.SubscriptionRepository,
	pkgRepo *repository.PackageRepository,
	userRepo *repository.UserRepository,
	auditRepo *repository.SubscriptionAuditRepository,
	razorpayClient *razorpay.Client,
	fileLogger *monitoring.FileAuditLogger,
	notifService *NotificationService,
) *SubscriptionService {
	return &SubscriptionService{
		subRepo:        subRepo,
		pkgRepo:        pkgRepo,
		userRepo:       userRepo,
		auditRepo:      auditRepo,
		razorpayClient: razorpayClient,
		fileLogger:     fileLogger,
		notifService:   notifService,
	}
}

// freeTierLimits returns free tier storage limits from DB, with hardcoded fallback.
func (s *SubscriptionService) freeTierLimits(ctx context.Context) (int64, int64) {
	freePkg, err := s.pkgRepo.FindByName(ctx, "free")
	if err == nil && freePkg != nil {
		return freePkg.StorageLimitBytes, freePkg.MaxFileSizeBytes
	}
	return model.DefaultFreeStorageLimitBytes, model.DefaultFreeMaxFileSizeBytes
}

// Subscribe initiates a subscription with Razorpay and creates a DB record with status 'created'.
// FIX #6: If DB insert fails after Razorpay subscription is created, we cancel the orphaned Razorpay sub.
func (s *SubscriptionService) Subscribe(ctx context.Context, userID, packageID string) (*model.Subscription, error) {
	// 1. Check for any active or pending subscription
	currentSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check current subscription: %w", err)
	}
	if currentSub != nil {
		if currentSub.Status == model.SubscriptionStatusCreated {
			// Unpaid or interrupted checkout attempt — cancel old one and allow re-subscribing
			if s.razorpayClient != nil && currentSub.RazorpaySubscriptionID != nil {
				_, _ = s.razorpayClient.CancelSubscription(ctx, *currentSub.RazorpaySubscriptionID, false)
			}
			now := time.Now().UTC()
			_ = s.subRepo.UpdateStatus(ctx, currentSub.ID, model.SubscriptionStatusCancelled, nil, nil, nil)
			_ = s.subRepo.SetCancellation(ctx, currentSub.ID, false, &now)
		} else {
			return nil, ErrActiveSubscriptionExists
		}
	}

	// 2. Fetch package
	pkg, err := s.pkgRepo.FindByID(ctx, packageID)
	if err != nil {
		return nil, fmt.Errorf("package not found: %w", err)
	}

	if pkg.PricePaise == 0 || pkg.RazorpayPlanID == nil || *pkg.RazorpayPlanID == "" {
		return nil, ErrFreeTierNoPayment
	}

	if s.razorpayClient == nil {
		return nil, errors.New("razorpay is not configured on server")
	}

	// 2b. Register or retrieve customer in Razorpay for saved cards and personalized checkout
	var rzpCustID *string
	user, userErr := s.userRepo.FindByID(ctx, userID)
	if userErr == nil && user != nil {
		firstName := ""
		if user.FirstName != nil {
			firstName = *user.FirstName
		}
		lastName := ""
		if user.LastName != nil {
			lastName = *user.LastName
		}
		fullName := strings.TrimSpace(firstName + " " + lastName)
		if fullName == "" {
			fullName = user.Email
		}
		cust, custErr := s.razorpayClient.CreateCustomer(ctx, fullName, user.Email, "")
		if custErr == nil && cust != nil && cust.ID != "" {
			rzpCustID = &cust.ID
		} else {
			log.Warn().Err(custErr).Str("email", user.Email).Msg("Could not register Razorpay customer; continuing with standard subscription")
		}
	}

	var custIDStr string
	if rzpCustID != nil {
		custIDStr = *rzpCustID
	}

	// 3. Create Razorpay Subscription
	rzpReq := razorpay.CreateSubscriptionRequest{
		PlanID:         *pkg.RazorpayPlanID,
		TotalCount:     120, // 10 years monthly cycles
		Quantity:       1,
		CustomerNotify: 1,
		CustomerID:     custIDStr,
		Notes: map[string]string{
			"user_id":    userID,
			"package_id": pkg.ID,
		},
	}

	rzpResp, err := s.razorpayClient.CreateSubscription(ctx, rzpReq)
	if err != nil {
		s.logAudit(ctx, userID, nil, nil, "subscription.initiation_failed", "api", "error", err.Error(), nil)
		return nil, fmt.Errorf("razorpay create subscription: %w", err)
	}

	// 4. Save subscription record in DB
	sub := &model.Subscription{
		UserID:                 userID,
		PackageID:              pkg.ID,
		RazorpaySubscriptionID: &rzpResp.ID,
		RazorpayCustomerID:     rzpCustID,
		Status:                 model.SubscriptionStatusCreated,
		Metadata: map[string]interface{}{
			"short_url": rzpResp.ShortURL,
		},
	}

	createdSub, err := s.subRepo.Create(ctx, sub)
	if err != nil {
		// CRITICAL: Cancel the orphaned Razorpay subscription to prevent untracked charges
		if _, cancelErr := s.razorpayClient.CancelSubscription(ctx, rzpResp.ID, false); cancelErr != nil {
			log.Error().Err(cancelErr).Str("rzp_sub_id", rzpResp.ID).
				Msg("CRITICAL: Failed to cancel orphaned Razorpay subscription after DB insert failure")
		}
		return nil, fmt.Errorf("create subscription record: %w", err)
	}
	createdSub.Package = pkg

	s.logAudit(ctx, userID, &createdSub.ID, nil, "subscription.created", "api", "success", "", map[string]interface{}{
		"razorpay_subscription_id": rzpResp.ID,
		"package_name":             pkg.Name,
	})

	return createdSub, nil
}

// Verify verifies the Razorpay signature after checkout modal completion and activates limits.
// FIX #1: Idempotency guard — if already activated by webhook, return current state without re-processing.
func (s *SubscriptionService) Verify(ctx context.Context, userID, rzpSubscriptionID, rzpPaymentID, rzpSignature string) (*model.Subscription, error) {
	if s.razorpayClient == nil {
		return nil, errors.New("razorpay client not initialized")
	}

	// 1. Verify HMAC Signature
	if !s.razorpayClient.VerifySubscriptionPaymentSignature(rzpPaymentID, rzpSubscriptionID, rzpSignature) {
		s.logAudit(ctx, userID, nil, nil, "subscription.verify_failed", "api", "error", "invalid payment signature", map[string]interface{}{
			"razorpay_subscription_id": rzpSubscriptionID,
			"razorpay_payment_id":      rzpPaymentID,
		})
		return nil, errors.New("invalid razorpay payment signature")
	}

	// 2. Fetch subscription record
	sub, err := s.subRepo.FindByRazorpayID(ctx, rzpSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("find subscription by razorpay id: %w", err)
	}
	if sub.UserID != userID {
		return nil, errors.New("subscription does not belong to user")
	}

	// 3. Idempotency: if already activated by webhook, return current state
	if sub.IsActive() {
		s.logAudit(ctx, userID, &sub.ID, nil, "subscription.verify_idempotent", "api", "success", "already active via webhook", nil)
		return sub, nil
	}

	// 4. Fetch latest state from Razorpay
	rzpSub, err := s.razorpayClient.FetchSubscription(ctx, rzpSubscriptionID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch fresh state from Razorpay, using active default")
	}

	var start, end *time.Time
	if rzpSub != nil && rzpSub.CurrentStart > 0 {
		st := time.Unix(rzpSub.CurrentStart, 0).UTC()
		start = &st
	}
	if rzpSub != nil && rzpSub.CurrentEnd > 0 {
		en := time.Unix(rzpSub.CurrentEnd, 0).UTC()
		end = &en
	}
	if start == nil {
		now := time.Now().UTC()
		start = &now
	}
	if end == nil {
		exp := time.Now().UTC().AddDate(0, 1, 0)
		end = &exp
	}

	status := model.SubscriptionStatusActive
	if rzpSub != nil && rzpSub.Status != "" && rzpSub.Status != model.SubscriptionStatusCreated {
		status = rzpSub.Status
	}

	// 5. Update subscription
	if err := s.subRepo.UpdateStatus(ctx, sub.ID, status, start, end, &rzpPaymentID); err != nil {
		return nil, fmt.Errorf("update subscription status: %w", err)
	}

	// 6. Update user storage limits
	pkg, err := s.pkgRepo.FindByID(ctx, sub.PackageID)
	if err == nil && pkg != nil {
		if limitErr := s.userRepo.UpdateStorageLimits(ctx, userID, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes); limitErr != nil {
			log.Error().Err(limitErr).Str("user_id", userID).Msg("Failed to update storage limits after verify")
		}
	}

	s.logAudit(ctx, userID, &sub.ID, nil, "subscription.verified", "api", "success", "", map[string]interface{}{
		"razorpay_subscription_id": rzpSubscriptionID,
		"razorpay_payment_id":      rzpPaymentID,
		"package_id":               sub.PackageID,
	})

	// 7. Notify user
	if s.notifService != nil && pkg != nil {
		_ = s.notifService.NotifySubscription(
			ctx, userID,
			"Subscription Activated!",
			fmt.Sprintf("Your ByteVault %s subscription is now active.", pkg.DisplayName),
			"subscription.activated",
		)
	}

	return s.subRepo.FindByID(ctx, sub.ID)
}

// Upgrade upgrades the user's active subscription immediately to a higher tier.
// FIX #2: Validates target is strictly higher tier by price. Clears pending downgrades/cancellations.
func (s *SubscriptionService) Upgrade(ctx context.Context, userID, targetPackageID string) (*model.Subscription, error) {
	currentSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil || currentSub == nil || !currentSub.IsActive() {
		return nil, ErrNoActiveSubscription
	}

	// Fetch current package for tier comparison
	currentPkg, err := s.pkgRepo.FindByID(ctx, currentSub.PackageID)
	if err != nil {
		return nil, fmt.Errorf("current package not found: %w", err)
	}

	targetPkg, err := s.pkgRepo.FindByID(ctx, targetPackageID)
	if err != nil {
		return nil, fmt.Errorf("target package not found: %w", err)
	}

	// Exploit guard: Cannot upgrade while a downgrade or cancellation is already scheduled
	if currentSub.PendingPackageID != nil || currentSub.CancelAtCycleEnd {
		return nil, fmt.Errorf("%w: a plan change or cancellation is already pending for this cycle", ErrInvalidPlanTransition)
	}

	// Tier validation: cannot upgrade to same or cheaper plan
	if targetPkg.ID == currentPkg.ID {
		return nil, ErrSamePlan
	}
	if targetPkg.PricePaise <= currentPkg.PricePaise {
		return nil, fmt.Errorf("%w: cannot upgrade to a cheaper or equal plan, use downgrade instead", ErrInvalidPlanTransition)
	}

	if targetPkg.RazorpayPlanID == nil || *targetPkg.RazorpayPlanID == "" {
		return nil, errors.New("target package has no razorpay plan")
	}

	// Upgrade via Razorpay API (immediate)
	if s.razorpayClient != nil && currentSub.RazorpaySubscriptionID != nil {
		_, err := s.razorpayClient.UpdateSubscription(ctx, *currentSub.RazorpaySubscriptionID, *targetPkg.RazorpayPlanID, "now")
		if err != nil {
			s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.upgrade_failed", "api", "error", err.Error(), nil)
			return nil, fmt.Errorf("razorpay upgrade failed: %w", err)
		}
	}

	// FIX #16: Log critical inconsistency if DB update fails after Razorpay already committed
	if err := s.subRepo.UpdatePackageID(ctx, currentSub.ID, targetPkg.ID); err != nil {
		log.Error().Err(err).Str("sub_id", currentSub.ID).Str("target_pkg", targetPkg.ID).
			Msg("CRITICAL: DB package update failed after Razorpay upgrade succeeded — manual reconciliation needed")
		return nil, fmt.Errorf("update subscription package: %w", err)
	}

	// Anti-ping-pong lock: Mark that user upgraded during this cycle
	_ = s.subRepo.SetUpgradedFrom(ctx, currentSub.ID, &currentPkg.ID)

	// Clear any pending downgrade since user is upgrading
	if currentSub.PendingPackageID != nil {
		if clearErr := s.subRepo.SetPendingDowngrade(ctx, currentSub.ID, nil); clearErr != nil {
			log.Warn().Err(clearErr).Msg("Failed to clear pending downgrade after upgrade")
		}
	}

	// Clear any pending cancellation since user is actively upgrading
	if currentSub.CancelAtCycleEnd {
		if clearErr := s.subRepo.SetCancellation(ctx, currentSub.ID, false, nil); clearErr != nil {
			log.Warn().Err(clearErr).Msg("Failed to clear pending cancellation after upgrade")
		}
	}

	// Immediately elevate user storage limits
	if limitErr := s.userRepo.UpdateStorageLimits(ctx, userID, targetPkg.StorageLimitBytes, targetPkg.MaxFileSizeBytes); limitErr != nil {
		log.Error().Err(limitErr).Str("user_id", userID).Msg("Failed to update storage limits after upgrade")
	}

	s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.upgraded", "api", "success", "", map[string]interface{}{
		"old_package_id": currentSub.PackageID,
		"new_package_id": targetPkg.ID,
	})

	if s.notifService != nil {
		_ = s.notifService.NotifySubscription(
			ctx, userID,
			"Plan Upgraded!",
			fmt.Sprintf("You have upgraded to ByteVault %s. Your storage limits have been expanded immediately.", targetPkg.DisplayName),
			"subscription.upgraded",
		)
	}

	return s.subRepo.FindByID(ctx, currentSub.ID)
}

// Downgrade schedules a downgrade to take effect at the end of the current billing cycle.
// FIX #2: Validates target is strictly lower tier. FIX #9: Free tier has its own code path.
func (s *SubscriptionService) Downgrade(ctx context.Context, userID, targetPackageID string) (*model.Subscription, error) {
	currentSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil || currentSub == nil || !currentSub.IsActive() {
		return nil, ErrNoActiveSubscription
	}

	// Fetch current package for tier comparison
	currentPkg, err := s.pkgRepo.FindByID(ctx, currentSub.PackageID)
	if err != nil {
		return nil, fmt.Errorf("current package not found: %w", err)
	}

	targetPkg, err := s.pkgRepo.FindByID(ctx, targetPackageID)
	if err != nil {
		return nil, fmt.Errorf("target package not found: %w", err)
	}

	// Exploit guard: Only 1 scheduled plan change per billing cycle
	if currentSub.PendingPackageID != nil || currentSub.CancelAtCycleEnd {
		return nil, fmt.Errorf("%w: a plan change or cancellation is already scheduled for this subscription", ErrInvalidPlanTransition)
	}
	if currentSub.UpgradedFromID != nil {
		return nil, fmt.Errorf("%w: you have already upgraded during the current billing cycle. Downgrades may only be scheduled after the current cycle renews", ErrInvalidPlanTransition)
	}

	// Tier validation: cannot downgrade to same or more expensive plan
	if targetPkg.ID == currentPkg.ID {
		return nil, ErrSamePlan
	}
	if targetPkg.PricePaise >= currentPkg.PricePaise {
		return nil, fmt.Errorf("%w: cannot downgrade to a more expensive or equal plan, use upgrade instead", ErrInvalidPlanTransition)
	}

	// FIX #9: Downgrading to free tier — handle as scheduled cancellation with proper downgrade context
	if targetPkg.PricePaise == 0 {
		if s.razorpayClient != nil && currentSub.RazorpaySubscriptionID != nil {
			if _, cancelErr := s.razorpayClient.CancelSubscription(ctx, *currentSub.RazorpaySubscriptionID, true); cancelErr != nil {
				log.Warn().Err(cancelErr).Msg("Razorpay cancel for free downgrade failed")
			}
		}

		// Set cancel at cycle end but DON'T set cancelled_at — cancellation is scheduled, not completed
		if err := s.subRepo.SetCancellation(ctx, currentSub.ID, true, nil); err != nil {
			return nil, fmt.Errorf("set cancellation for free downgrade: %w", err)
		}
		// Track the pending free tier downgrade so scheduler applies correct limits
		if err := s.subRepo.SetPendingDowngrade(ctx, currentSub.ID, &targetPkg.ID); err != nil {
			return nil, fmt.Errorf("set pending downgrade to free: %w", err)
		}

		s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.downgrade_to_free_scheduled", "api", "success", "", map[string]interface{}{
			"target_package_id": targetPkg.ID,
		})

		if s.notifService != nil {
			cycleEnd := "the end of your billing cycle"
			if currentSub.CurrentPeriodEnd != nil {
				cycleEnd = currentSub.CurrentPeriodEnd.Format("02 Jan 2006")
			}
			_ = s.notifService.NotifySubscription(
				ctx, userID,
				"Downgrade to Free Scheduled",
				fmt.Sprintf("Your subscription will switch to the Free tier on %s. Your existing files will remain, but uploads will be limited.", cycleEnd),
				"subscription.downgrade_to_free_scheduled",
			)
		}

		return s.subRepo.FindByID(ctx, currentSub.ID)
	}

	// Paid-to-paid downgrade
	if targetPkg.RazorpayPlanID == nil || *targetPkg.RazorpayPlanID == "" {
		return nil, errors.New("target package has no razorpay plan")
	}

	// Schedule change in Razorpay at cycle end
	if s.razorpayClient != nil && currentSub.RazorpaySubscriptionID != nil {
		_, err := s.razorpayClient.UpdateSubscription(ctx, *currentSub.RazorpaySubscriptionID, *targetPkg.RazorpayPlanID, "cycle_end")
		if err != nil {
			return nil, fmt.Errorf("schedule razorpay downgrade: %w", err)
		}
	}

	if err := s.subRepo.SetPendingDowngrade(ctx, currentSub.ID, &targetPkg.ID); err != nil {
		return nil, fmt.Errorf("set pending downgrade: %w", err)
	}

	s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.downgrade_scheduled", "api", "success", "", map[string]interface{}{
		"target_package_id": targetPkg.ID,
	})

	if s.notifService != nil {
		cycleEnd := "the end of your billing cycle"
		if currentSub.CurrentPeriodEnd != nil {
			cycleEnd = currentSub.CurrentPeriodEnd.Format("02 Jan 2006")
		}
		_ = s.notifService.NotifySubscription(
			ctx, userID,
			"Downgrade Scheduled",
			fmt.Sprintf("Your subscription will switch to %s on %s.", targetPkg.DisplayName, cycleEnd),
			"subscription.downgrade_scheduled",
		)
	}

	return s.subRepo.FindByID(ctx, currentSub.ID)
}

// Cancel requests subscription cancellation at cycle end. Access is maintained until period end.
// FIX #7: Don't set cancelled_at — cancellation is scheduled, not completed. Webhook sets it when Razorpay confirms.
func (s *SubscriptionService) Cancel(ctx context.Context, userID string) (*model.Subscription, error) {
	currentSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil || currentSub == nil || !currentSub.IsActive() {
		return nil, ErrNoActiveSubscription
	}

	// Cancel on Razorpay at cycle end
	if s.razorpayClient != nil && currentSub.RazorpaySubscriptionID != nil {
		_, err := s.razorpayClient.CancelSubscription(ctx, *currentSub.RazorpaySubscriptionID, true)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to call razorpay cancel subscription, marking in DB anyway")
		}
	}

	// Only set cancel_at_cycle_end flag. DON'T set cancelled_at — cancellation is scheduled, not completed.
	if err := s.subRepo.SetCancellation(ctx, currentSub.ID, true, nil); err != nil {
		return nil, fmt.Errorf("set cancellation: %w", err)
	}

	s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.cancellation_requested", "api", "success", "", nil)

	if s.notifService != nil {
		cycleEnd := "the end of your billing period"
		if currentSub.CurrentPeriodEnd != nil {
			cycleEnd = currentSub.CurrentPeriodEnd.Format("02 Jan 2006")
		}
		_ = s.notifService.NotifySubscription(
			ctx, userID,
			"Cancellation Scheduled",
			fmt.Sprintf("Your subscription has been scheduled for cancellation. You will continue to have full access until %s.", cycleEnd),
			"subscription.cancellation_scheduled",
		)
	}

	return s.subRepo.FindByID(ctx, currentSub.ID)
}

// CancelPendingDowngrade revokes a scheduled plan downgrade, keeping the user on their current plan.
func (s *SubscriptionService) CancelPendingDowngrade(ctx context.Context, userID string) (*model.Subscription, error) {
	currentSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil || currentSub == nil || !currentSub.IsActive() {
		return nil, ErrNoActiveSubscription
	}

	if currentSub.PendingPackageID == nil {
		return nil, errors.New("no scheduled downgrade found for this subscription")
	}

	if err := s.subRepo.SetPendingDowngrade(ctx, currentSub.ID, nil); err != nil {
		return nil, fmt.Errorf("clear pending downgrade: %w", err)
	}

	// If cancellation flag was set for free-tier downgrade, clear it
	if currentSub.CancelAtCycleEnd {
		_ = s.subRepo.SetCancellation(ctx, currentSub.ID, false, nil)
	}

	s.logAudit(ctx, userID, &currentSub.ID, nil, "subscription.downgrade_cancelled", "api", "success", "", nil)

	if s.notifService != nil {
		_ = s.notifService.NotifySubscription(
			ctx, userID,
			"Downgrade Request Cancelled",
			"Your scheduled plan downgrade has been cancelled. Your current subscription will continue renewing as normal.",
			"subscription.downgrade_cancelled",
		)
	}

	return s.subRepo.FindByID(ctx, currentSub.ID)
}

// GetCurrentSubscription returns the user's current subscription (any non-terminal state) or a free tier fallback.
// FIX #13: Returns subscription in any in-progress state for frontend to render correctly (e.g. "processing payment").
func (s *SubscriptionService) GetCurrentSubscription(ctx context.Context, userID string) (*model.Subscription, error) {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.Status != model.SubscriptionStatusCreated {
		return sub, nil
	}

	// Fallback to Free Package
	freePkg, err := s.pkgRepo.FindByName(ctx, "free")
	if err != nil || freePkg == nil {
		freePkg = &model.Package{
			Name:              "free",
			DisplayName:       "Free",
			PricePaise:        0,
			StorageLimitBytes: model.DefaultFreeStorageLimitBytes,
			MaxFileSizeBytes:  model.DefaultFreeMaxFileSizeBytes,
		}
	}

	return &model.Subscription{
		UserID:    userID,
		PackageID: freePkg.ID,
		Status:    model.SubscriptionStatusActive,
		Package:   freePkg,
	}, nil
}

// AdminCancel immediately cancels a subscription and resets user limits to free tier.
// FIX #5: Propagates errors instead of silently discarding them.
func (s *SubscriptionService) AdminCancel(ctx context.Context, subscriptionID string) error {
	sub, err := s.subRepo.FindByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("find subscription: %w", err)
	}

	// Cancel on Razorpay (immediate, not at cycle end)
	if s.razorpayClient != nil && sub.RazorpaySubscriptionID != nil {
		if _, rzpErr := s.razorpayClient.CancelSubscription(ctx, *sub.RazorpaySubscriptionID, false); rzpErr != nil {
			log.Error().Err(rzpErr).Str("sub_id", subscriptionID).
				Msg("Razorpay cancel failed during admin cancel — proceeding with DB update")
		}
	}

	now := time.Now().UTC()
	if err := s.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusCancelled, nil, nil, nil); err != nil {
		return fmt.Errorf("update subscription status: %w", err)
	}
	if err := s.subRepo.SetCancellation(ctx, sub.ID, false, &now); err != nil {
		return fmt.Errorf("set cancellation timestamp: %w", err)
	}

	// Reset limits to Free tier
	storageLimit, maxFileSize := s.freeTierLimits(ctx)
	if err := s.userRepo.UpdateStorageLimits(ctx, sub.UserID, storageLimit, maxFileSize); err != nil {
		return fmt.Errorf("reset storage limits: %w", err)
	}

	s.logAudit(ctx, sub.UserID, &sub.ID, nil, "subscription.admin_cancelled", "admin", "success", "", nil)

	if s.notifService != nil {
		_ = s.notifService.NotifySubscription(
			ctx, sub.UserID,
			"Subscription Cancelled by Admin",
			"Your ByteVault subscription has been cancelled by an administrator. Your storage has been reset to the Free tier.",
			"subscription.admin_cancelled",
		)
	}

	return nil
}

// AdminAssign assigns a package to a user directly (bypassing payment).
// FIX #14: Cancels any existing active subscription before creating a new one.
func (s *SubscriptionService) AdminAssign(ctx context.Context, userID, packageID string, durationDays int) (*model.Subscription, error) {
	// Cancel any existing active or pending subscription to avoid constraint violation
	existingSub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing subscription: %w", err)
	}
	if existingSub != nil {
		if cancelErr := s.AdminCancel(ctx, existingSub.ID); cancelErr != nil {
			return nil, fmt.Errorf("cancel existing subscription before admin assign: %w", cancelErr)
		}
	}

	pkg, err := s.pkgRepo.FindByID(ctx, packageID)
	if err != nil {
		return nil, fmt.Errorf("package not found: %w", err)
	}

	start := time.Now().UTC()
	end := start.AddDate(0, 0, durationDays)

	sub := &model.Subscription{
		UserID:             userID,
		PackageID:          pkg.ID,
		Status:             model.SubscriptionStatusActive,
		CurrentPeriodStart: &start,
		CurrentPeriodEnd:   &end,
		Metadata: map[string]interface{}{
			"assigned_by": "admin",
		},
	}

	created, err := s.subRepo.Create(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("create assigned subscription: %w", err)
	}

	if limitErr := s.userRepo.UpdateStorageLimits(ctx, userID, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes); limitErr != nil {
		log.Error().Err(limitErr).Msg("Failed to update storage limits for admin assign")
	}

	s.logAudit(ctx, userID, &created.ID, nil, "subscription.admin_assigned", "admin", "success", "", map[string]interface{}{
		"package_name": pkg.Name,
		"days":         durationDays,
	})

	return created, nil
}

// SyncWithRazorpay fetches the latest subscription state from Razorpay and reconciles local DB.
// FIX #6 (user request): Provides a way to verify and fix subscription state mismatches.
func (s *SubscriptionService) SyncWithRazorpay(ctx context.Context, subscriptionID string) (*model.Subscription, error) {
	sub, err := s.subRepo.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("find subscription: %w", err)
	}

	if s.razorpayClient == nil || sub.RazorpaySubscriptionID == nil {
		return sub, nil
	}

	rzpSub, err := s.razorpayClient.FetchSubscription(ctx, *sub.RazorpaySubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("fetch from razorpay: %w", err)
	}

	var start, end *time.Time
	if rzpSub.CurrentStart > 0 {
		st := time.Unix(rzpSub.CurrentStart, 0).UTC()
		start = &st
	}
	if rzpSub.CurrentEnd > 0 {
		en := time.Unix(rzpSub.CurrentEnd, 0).UTC()
		end = &en
	}

	if err := s.subRepo.UpdateStatus(ctx, sub.ID, rzpSub.Status, start, end, nil); err != nil {
		return nil, fmt.Errorf("sync status update: %w", err)
	}

	// If Razorpay shows active, ensure storage limits match package
	if rzpSub.Status == model.SubscriptionStatusActive {
		pkg, pkgErr := s.pkgRepo.FindByID(ctx, sub.PackageID)
		if pkgErr == nil && pkg != nil {
			_ = s.userRepo.UpdateStorageLimits(ctx, sub.UserID, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes)
		}
	}

	s.logAudit(ctx, sub.UserID, &sub.ID, nil, "subscription.synced", "api", "success", "", map[string]interface{}{
		"razorpay_status": rzpSub.Status,
	})

	return s.subRepo.FindByID(ctx, sub.ID)
}

// ProcessScheduledDowngrades is run by the scheduler to apply downgrades and cancellations whose period ended.
// FIX #11: Handles orphaned pending_package_id by clearing it and notifying user when target package is unavailable.
func (s *SubscriptionService) ProcessScheduledDowngrades(ctx context.Context) error {
	now := time.Now().UTC()

	// 1. Process pending downgrades (paid-to-paid and paid-to-free)
	downgrades, err := s.subRepo.ListPendingDowngrades(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list pending downgrades")
	} else {
		for _, sub := range downgrades {
			if sub.PendingPackageID == nil {
				continue
			}
			targetPkg, pkgErr := s.pkgRepo.FindByID(ctx, *sub.PendingPackageID)
			if pkgErr != nil || targetPkg == nil {
				// Target package no longer exists — clear pending downgrade and notify user
				log.Error().Str("sub_id", sub.ID).Str("pending_pkg_id", *sub.PendingPackageID).
					Msg("Pending downgrade package not found, clearing")
				_ = s.subRepo.SetPendingDowngrade(ctx, sub.ID, nil)
				if s.notifService != nil {
					_ = s.notifService.NotifySubscription(
						ctx, sub.UserID,
						"Downgrade Failed",
						"Your scheduled plan downgrade could not be applied because the target plan is no longer available. Your current plan remains active.",
						"subscription.downgrade_failed",
					)
				}
				s.logAudit(ctx, sub.UserID, &sub.ID, nil, "subscription.downgrade_failed", "scheduler", "error", "target package not found", nil)
				continue
			}

			if applyErr := s.subRepo.UpdatePackageID(ctx, sub.ID, targetPkg.ID); applyErr != nil {
				log.Error().Err(applyErr).Str("sub_id", sub.ID).Msg("Failed to apply scheduled downgrade")
				continue
			}
			if limitErr := s.userRepo.UpdateStorageLimits(ctx, sub.UserID, targetPkg.StorageLimitBytes, targetPkg.MaxFileSizeBytes); limitErr != nil {
				log.Error().Err(limitErr).Str("sub_id", sub.ID).Msg("Failed to update storage limits for downgrade")
			}

			s.logAudit(ctx, sub.UserID, &sub.ID, nil, "subscription.downgrade_applied", "scheduler", "success", "", map[string]interface{}{
				"new_package_id": targetPkg.ID,
			})

			if s.notifService != nil {
				_ = s.notifService.NotifySubscription(
					ctx, sub.UserID,
					"Plan Changed",
					fmt.Sprintf("Your ByteVault subscription has been changed to %s as scheduled.", targetPkg.DisplayName),
					"subscription.downgrade_applied",
				)
			}
		}
	}

	// 2. Process expired cancellations (active subs with cancel_at_cycle_end past their period)
	//    and cancelled subs whose period has ended (webhook set cancelled but period was in the future)
	expiring, err := s.subRepo.ListExpiringOrExpired(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list expiring subscriptions")
	} else {
		storageLimit, maxFileSize := s.freeTierLimits(ctx)
		for _, sub := range expiring {
			_ = s.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionStatusExpired, nil, nil, nil)
			_ = s.userRepo.UpdateStorageLimits(ctx, sub.UserID, storageLimit, maxFileSize)
			s.logAudit(ctx, sub.UserID, &sub.ID, nil, "subscription.expired_and_downgraded", "scheduler", "success", "", nil)

			if s.notifService != nil {
				_ = s.notifService.NotifySubscription(
					ctx, sub.UserID,
					"Subscription Expired",
					"Your subscription period has ended. Your storage has been reset to the Free tier.",
					"subscription.expired",
				)
			}
		}
	}

	// 3. Dispatch renewal reminders for subscriptions renewing in 7, 5, 3, 2, or 1 days
	reminderMilestones := []int{7, 5, 3, 2, 1}
	renewals, err := s.subRepo.ListUpcomingRenewals(ctx, now, now.Add(8*24*time.Hour))
	if err == nil && s.notifService != nil {
		for _, sub := range renewals {
			if sub.CurrentPeriodEnd == nil || sub.CancelAtCycleEnd {
				continue
			}

			diffHours := sub.CurrentPeriodEnd.Sub(now).Hours()
			if diffHours <= 0 {
				continue
			}
			daysRemaining := int(math.Ceil(diffHours / 24.0))

			for _, targetDay := range reminderMilestones {
				if daysRemaining == targetDay {
					if sub.Metadata == nil {
						sub.Metadata = make(map[string]interface{})
					}

					key := fmt.Sprintf("reminder_%dd_sent", targetDay)
					if sent, ok := sub.Metadata[key].(bool); ok && sent {
						continue // Already sent for this milestone
					}

					pkgName := "Pro"
					if sub.Package != nil {
						pkgName = sub.Package.DisplayName
					}
					renewalDate := sub.CurrentPeriodEnd.Format("02 Jan 2006")

					dayText := fmt.Sprintf("%d days", targetDay)
					if targetDay == 1 {
						dayText = "1 day"
					}

					_ = s.notifService.NotifySubscription(
						ctx, sub.UserID,
						fmt.Sprintf("Subscription Renewal in %s", dayText),
						fmt.Sprintf("Your ByteVault %s subscription is scheduled to renew in %s on %s. Please ensure your payment method has sufficient balance.", pkgName, dayText, renewalDate),
						"subscription.renewal_reminder",
					)

					sub.Metadata[key] = true
					_ = s.subRepo.UpdateMetadata(ctx, sub.ID, sub.Metadata)

					s.logAudit(ctx, sub.UserID, &sub.ID, nil,
						"subscription.renewal_reminder_sent", "scheduler", "success", "",
						map[string]interface{}{
							"days_remaining":     targetDay,
							"current_period_end": sub.CurrentPeriodEnd,
						},
					)
					break
				}
			}
		}
	}

	return nil
}

// logAudit writes to both DB and file audit logs.
// FIX #19: Logs warnings instead of silently discarding audit write failures.
func (s *SubscriptionService) logAudit(
	ctx context.Context,
	userID string,
	subID, txnID *string,
	eventType, eventSource, status, errMsg string,
	payload map[string]interface{},
) {
	entry := &model.SubscriptionAuditLog{
		UserID:         &userID,
		SubscriptionID: subID,
		TransactionID:  txnID,
		EventType:      eventType,
		EventSource:    eventSource,
		Status:         status,
		Payload:        payload,
	}
	if errMsg != "" {
		entry.ErrorMessage = &errMsg
	}
	if dbErr := s.auditRepo.Log(ctx, entry); dbErr != nil {
		log.Warn().Err(dbErr).Str("event_type", eventType).Msg("Failed to write audit log to database")
	}

	// Also write to daily JSONL file
	if s.fileLogger != nil {
		subStr := ""
		if subID != nil {
			subStr = *subID
		}
		txnStr := ""
		if txnID != nil {
			txnStr = *txnID
		}
		if fileErr := s.fileLogger.Log(monitoring.AuditEvent{
			EventType:      eventType,
			EventSource:    eventSource,
			UserID:         userID,
			SubscriptionID: subStr,
			TransactionID:  txnStr,
			Status:         status,
			ErrorMessage:   errMsg,
			Payload:        payload,
		}); fileErr != nil {
			log.Warn().Err(fileErr).Str("event_type", eventType).Msg("Failed to write audit log to file")
		}
	}
}
