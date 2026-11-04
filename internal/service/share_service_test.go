package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
)

type mockUserFinder struct {
	user *model.User
	err  error
}

func (m *mockUserFinder) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserFinder) FindByID(ctx context.Context, id string) (*model.User, error) {
	return m.user, m.err
}

type mockShareStore struct {
	shares               []*model.Share
	listSharedWithMeCalls int
}

func (m *mockShareStore) Create(ctx context.Context, share *model.Share) error {
	return nil
}
func (m *mockShareStore) ListByResource(ctx context.Context, resourceType, resourceID, ownerID string) ([]*model.Share, error) {
	return nil, nil
}
func (m *mockShareStore) ListSharedWithMe(ctx context.Context, granteeEmail string) ([]*model.Share, error) {
	m.listSharedWithMeCalls++
	return m.shares, nil
}
func (m *mockShareStore) FindByID(ctx context.Context, id string) (*model.Share, error) {
	return nil, nil
}
func (m *mockShareStore) FindByResourceAndGrantee(ctx context.Context, resourceType, resourceID, granteeEmail string) (*model.Share, error) {
	return nil, nil
}
func (m *mockShareStore) Revoke(ctx context.Context, id, ownerID string) error {
	return nil
}

func TestShareService_UnverifiedUserIsolation(t *testing.T) {
	ctx := context.Background()

	// 1. Unverified user account (is_verified = false)
	unverifiedUser := &model.User{
		ID:         "unverified-u1",
		Email:      "unverified@example.com",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}
	userFinderUnverified := &mockUserFinder{user: unverifiedUser}
	shareStoreUnverified := &mockShareStore{
		shares: []*model.Share{
			{ID: "share-1", GranteeEmail: "unverified@example.com", Permission: "VIEWER"},
		},
	}

	shareSvcUnverified := service.NewShareService(
		shareStoreUnverified,
		userFinderUnverified,
		nil, nil, nil,
	)

	// ListSharedWithMe MUST return empty slice immediately for unverified user
	shares, err := shareSvcUnverified.ListSharedWithMe(ctx, "unverified@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 0 {
		t.Fatalf("expected 0 shares for unverified user, got %d", len(shares))
	}
	if shareStoreUnverified.listSharedWithMeCalls != 0 {
		t.Fatalf("expected repository ListSharedWithMe NOT to be called for unverified user, got %d calls", shareStoreUnverified.listSharedWithMeCalls)
	}

	// ListSharedWithMeByUserID MUST also return empty slice
	sharesByUID, err := shareSvcUnverified.ListSharedWithMeByUserID(ctx, "unverified-u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sharesByUID) != 0 {
		t.Fatalf("expected 0 shares for unverified user by UID, got %d", len(sharesByUID))
	}

	// CheckAccessPermission MUST return false for unverified grantee
	hasAccess, err := shareSvcUnverified.CheckAccessPermission(ctx, "unverified-u1", "unverified@example.com", "file", "file-123", "VIEWER")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasAccess {
		t.Fatal("expected unverified user to have access denied (false), got true")
	}

	// 2. Verified user account (is_verified = true)
	verifiedUser := &model.User{
		ID:         "verified-u2",
		Email:      "verified@example.com",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}
	userFinderVerified := &mockUserFinder{user: verifiedUser}
	shareStoreVerified := &mockShareStore{
		shares: []*model.Share{
			{ID: "share-1", GranteeEmail: "verified@example.com", Permission: "VIEWER"},
		},
	}

	shareSvcVerified := service.NewShareService(
		shareStoreVerified,
		userFinderVerified,
		nil, nil, nil,
	)

	sharesVerified, err := shareSvcVerified.ListSharedWithMe(ctx, "verified@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sharesVerified) != 1 {
		t.Fatalf("expected 1 share for verified user, got %d", len(sharesVerified))
	}
	if shareStoreVerified.listSharedWithMeCalls != 1 {
		t.Fatalf("expected repository ListSharedWithMe to be called once for verified user, got %d calls", shareStoreVerified.listSharedWithMeCalls)
	}
}
