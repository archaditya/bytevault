package service_test

import (
	"context"
	"testing"

	"github.com/archaditya/bytevault/internal/service"
)

func TestConsentServiceValidation(t *testing.T) {
	svc := service.NewConsentService(nil)

	// Missing user ID
	_, err := svc.RecordGrant(context.Background(), "", service.ConsentTypeTermsAndPrivacy, "2026.1", "127.0.0.1", "test-agent")
	if err != service.ErrConsentUserRequired {
		t.Fatalf("expected ErrConsentUserRequired, got %v", err)
	}

	// Missing consent type
	_, err = svc.RecordGrant(context.Background(), "user-123", "", "2026.1", "127.0.0.1", "test-agent")
	if err != service.ErrConsentTypeRequired {
		t.Fatalf("expected ErrConsentTypeRequired, got %v", err)
	}

	// Revoke missing args
	err = svc.Revoke(context.Background(), "", "marketing")
	if err == nil {
		t.Fatal("expected error on empty user_id for revoke, got nil")
	}

	// Get consents missing args
	_, err = svc.GetUserConsents(context.Background(), "")
	if err != service.ErrConsentUserRequired {
		t.Fatalf("expected ErrConsentUserRequired on GetUserConsents, got %v", err)
	}
}
