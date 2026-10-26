package service

import (
	"context"
	"errors"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

var (
	ErrConsentTypeRequired = errors.New("consent_type is required")
	ErrConsentUserRequired = errors.New("user_id is required")
)

const (
	CurrentPolicyVersion = "2026.1"
	ConsentTypeTermsAndPrivacy = "terms_and_privacy"
	ConsentTypeMarketing       = "marketing"
)

type ConsentService struct {
	repo *repository.ConsentRepository
}

func NewConsentService(repo *repository.ConsentRepository) *ConsentService {
	return &ConsentService{repo: repo}
}

// RecordGrant records an affirmative consent action with version and client context.
func (s *ConsentService) RecordGrant(ctx context.Context, userID, consentType, version, ip, userAgent string) (*model.UserConsent, error) {
	if userID == "" {
		return nil, ErrConsentUserRequired
	}
	if consentType == "" {
		return nil, ErrConsentTypeRequired
	}
	if version == "" {
		version = CurrentPolicyVersion
	}

	consent := &model.UserConsent{
		UserID:      userID,
		ConsentType: consentType,
		Version:     version,
		Status:      "granted",
		GrantedAt:   time.Now(),
	}
	if ip != "" {
		consent.IPAddress = &ip
	}
	if userAgent != "" {
		consent.UserAgent = &userAgent
	}

	if err := s.repo.RecordConsent(ctx, consent); err != nil {
		return nil, err
	}
	return consent, nil
}

// Revoke updates the active consent of a given type to revoked.
func (s *ConsentService) Revoke(ctx context.Context, userID, consentType string) error {
	if userID == "" || consentType == "" {
		return errors.New("user_id and consent_type are required")
	}
	return s.repo.RevokeConsent(ctx, userID, consentType)
}

// GetUserConsents fetches all recorded consent audit items for a user.
func (s *ConsentService) GetUserConsents(ctx context.Context, userID string) ([]*model.UserConsent, error) {
	if userID == "" {
		return nil, ErrConsentUserRequired
	}
	return s.repo.GetConsentsByUserID(ctx, userID)
}
