package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
)

func TestAbuseService_Validation(t *testing.T) {
	svc := service.NewAbuseService(nil, nil, nil, nil)
	ctx := context.Background()

	// Missing reporter name
	_, err := svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "",
		ReporterEmail: "agent@copyright.com",
		Category:      "copyright",
		TargetURL:     "https://pushpostvault.com/s/123e4567-e89b-12d3-a456-426614174000",
		Description:   "Infringing material",
	})
	if err == nil || !strings.Contains(err.Error(), "reporter_name is required") {
		t.Fatalf("expected reporter_name error, got %v", err)
	}

	// Missing reporter email
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "",
		Category:      "copyright",
		TargetURL:     "https://pushpostvault.com/s/123e4567-e89b-12d3-a456-426614174000",
		Description:   "Infringing material",
	})
	if err == nil || !strings.Contains(err.Error(), "reporter_email is required") {
		t.Fatalf("expected reporter_email error, got %v", err)
	}

	// Missing category
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "agent@copyright.com",
		Category:      "",
		TargetURL:     "https://pushpostvault.com/s/123e4567-e89b-12d3-a456-426614174000",
		Description:   "Infringing material",
	})
	if err == nil || !strings.Contains(err.Error(), "category is required") {
		t.Fatalf("expected category error, got %v", err)
	}

	// Missing target_url
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "agent@copyright.com",
		Category:      "copyright",
		TargetURL:     "",
		Description:   "Infringing material",
	})
	if err == nil || !strings.Contains(err.Error(), "target_url is required") {
		t.Fatalf("expected target_url error, got %v", err)
	}

	// Instant share rejection
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "agent@copyright.com",
		Category:      "copyright",
		TargetURL:     "https://pushpostvault.com/e/instant/309584654f49a31129ca2adb0315fb5f",
		Description:   "Infringing material",
	})
	if err == nil || !strings.Contains(err.Error(), "Instant Shares expire automatically") {
		t.Fatalf("expected instant share rejection, got %v", err)
	}

	// Non-UUID invalid link rejection
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "agent@copyright.com",
		Category:      "copyright",
		TargetURL:     "https://google.com/search?q=test",
		Description:   "Infringing material",
	})
	if err == nil || (!strings.Contains(err.Error(), "Invalid shared link format") && !strings.Contains(err.Error(), "Only PushPostVault public shared links")) {
		t.Fatalf("expected invalid link format error, got %v", err)
	}

	// Missing description
	_, err = svc.SubmitReport(ctx, &model.SubmitAbuseReportRequest{
		ReporterName:  "Agent Smith",
		ReporterEmail: "agent@copyright.com",
		Category:      "copyright",
		TargetURL:     "https://pushpostvault.com/s/123e4567-e89b-12d3-a456-426614174000",
		Description:   "",
	})
	if err == nil || !strings.Contains(err.Error(), "description is required") {
		t.Fatalf("expected description error, got %v", err)
	}
}
