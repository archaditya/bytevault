package service_test

import (
	"context"
	"testing"

	"github.com/archaditya/bytevault/internal/service"
)

func TestFolderService_GetOrCreatePath_EmptySegments(t *testing.T) {
	svc := service.NewFolderService(nil, nil, nil)
	ctx := context.Background()

	// An empty slice of segments should return nil without attempting repository calls
	folder, err := svc.GetOrCreatePath(ctx, "user-123", []string{})
	if err != nil {
		t.Fatalf("expected nil error on empty segments, got: %v", err)
	}
	if folder != nil {
		t.Fatalf("expected nil folder on empty segments, got: %v", folder)
	}

	// Slices containing only whitespace or blank segments should also return nil safely
	folder2, err := svc.GetOrCreatePath(ctx, "user-123", []string{"", "   ", "\t"})
	if err != nil {
		t.Fatalf("expected nil error on blank segments, got: %v", err)
	}
	if folder2 != nil {
		t.Fatalf("expected nil folder on blank segments, got: %v", folder2)
	}
}
