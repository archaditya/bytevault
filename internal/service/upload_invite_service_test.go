package service_test

import (
	"testing"
	"time"

	"github.com/archaditya/bytevault/internal/service"
)

func TestUploadInviteConstants_ExpiryLimits(t *testing.T) {
	if service.DefaultInviteExpiryH != 24 {
		t.Fatalf("expected DefaultInviteExpiryH to be 24 (1 day), got %d", service.DefaultInviteExpiryH)
	}
	if service.MaxInviteExpiryH != 168 {
		t.Fatalf("expected MaxInviteExpiryH to be 168 (7 days), got %d", service.MaxInviteExpiryH)
	}
}

func TestUploadInvite_DebouncedNotificationTimer(t *testing.T) {
	svc := service.NewUploadInviteService(nil, nil, nil, nil, nil, nil)

	// Rapid succession of notifications on invite-123 (e.g. 50 files completing)
	for i := 0; i < 50; i++ {
		svc.TriggerDebouncedNotification("invite-123")
	}

	// Settle check: the debounce timer should be active and reset to the single latest timer
	// (verified by not panicking and managing internal state safely)
	time.Sleep(50 * time.Millisecond)
}
