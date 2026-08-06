package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

type ShareService struct {
	shareRepo    *repository.ShareRepository
	userRepo     *repository.UserRepository
	fileRepo     *repository.FileRepository
	folderRepo   *repository.FolderRepository
	activityRepo *repository.ActivityRepository
}

func NewShareService(
	shareRepo *repository.ShareRepository,
	userRepo *repository.UserRepository,
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	activityRepo *repository.ActivityRepository,
) *ShareService {
	return &ShareService{
		shareRepo:    shareRepo,
		userRepo:     userRepo,
		fileRepo:     fileRepo,
		folderRepo: folderRepo,
		activityRepo: activityRepo,
	}
}

func (s *ShareService) logActivity(ctx context.Context, userID, action, resourceID string, meta map[string]any) {
	if s.activityRepo == nil {
		return
	}

	resType := "share"

	_ = s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: &resType,
		ResourceID:   &resourceID,
		Metadata:     meta,
	})
}

// GrantShare verifies resource ownership and grants permission to grantee email
func (s *ShareService) GrantShare(ctx context.Context, ownerID, resourceType, resourceID, granteeEmail, permission string) (*model.Share, error) {
	granteeEmail = strings.ToLower(strings.TrimSpace(granteeEmail))
	permission = strings.ToUpper(strings.TrimSpace(permission))

	if permission != "VIEWER" && permission != "EDITOR" {
		permission = "VIEWER"
	}

	// Verify caller owns the resource
	if resourceType == "file" {
		file, err := s.fileRepo.FindByID(ctx, resourceID)
		if err != nil || file == nil || file.UserID != ownerID {
			return nil, fmt.Errorf("file not found or unauthorized")
		}
	} else if resourceType == "folder" {
		folder, err := s.folderRepo.FindByID(ctx, resourceID)
		if err != nil || folder == nil || folder.UserID != ownerID {
			return nil, fmt.Errorf("folder not found or unauthorized")
		}
	} else {
		return nil, fmt.Errorf("invalid resource_type: must be 'file' or 'folder'")
	}

	// Prevent self-sharing
	owner, err := s.userRepo.FindByID(ctx, ownerID)
	if err == nil && owner != nil && strings.ToLower(owner.Email) == granteeEmail {
		return nil, errors.New("cannot share a resource with yourself")
	}

	// Resolve grantee user ID if they're a registered user
	var granteeID *string
	granteeUser, _ := s.userRepo.FindByEmail(ctx, granteeEmail)
	if granteeUser != nil {
		granteeID = &granteeUser.ID
	}

	share := &model.Share{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OwnerID:      ownerID,
		GranteeEmail: granteeEmail,
		GranteeID:    granteeID,
		Permission:   permission,
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to grant share: %w", err)
	}

	s.logActivity(ctx, ownerID, "share.created", resourceID, map[string]any{
		"resource_type": resourceType,
		"grantee_email": granteeEmail,
		"permission":    permission,
	})

	return share, nil
}

func (s *ShareService) ListResourceShares(ctx context.Context, ownerID, resourceType, resourceID string) ([]*model.Share, error) {
	return s.shareRepo.ListByResource(ctx, resourceType, resourceID, ownerID)
}

func (s *ShareService) ListSharedWithMe(ctx context.Context, userEmail string) ([]*model.Share, error) {
	return s.shareRepo.ListSharedWithMe(ctx, userEmail)
}

func (s *ShareService) RevokeShare(ctx context.Context, ownerID, shareID string) error {
	share, err := s.shareRepo.FindByID(ctx, shareID)
	if err != nil {
		return err
	}

	if share.OwnerID != ownerID {
		return errors.New("unauthorized")
	}

	s.logActivity(ctx, ownerID, "share.revoked", share.ResourceID, map[string]any{
		"resource_type": share.ResourceType,
		"grantee_email": share.GranteeEmail,
		"permission":    share.Permission,
	})
	return s.shareRepo.Revoke(ctx, shareID, ownerID)
}

// CheckAccessPermission is the central security gate for granular sharing
func (s *ShareService) CheckAccessPermission(ctx context.Context, userID, userEmail, resourceType, resourceID, requiredPerm string) (bool, error) {
	// Owner always has full access
	if resourceType == "file" {
		file, err := s.fileRepo.FindByID(ctx, resourceID)
		if err == nil && file != nil && file.UserID == userID {
			return true, nil
		}
	} else if resourceType == "folder" {
		folder, err := s.folderRepo.FindByID(ctx, resourceID)
		if err == nil && folder != nil && folder.UserID == userID {
			return true, nil
		}
	}

	// Check shares table
	share, err := s.shareRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, userEmail)
	if err != nil || share == nil {
		return false, nil
	}

	if requiredPerm == "VIEWER" {
		return share.Permission == "VIEWER" || share.Permission == "EDITOR", nil
	}
	if requiredPerm == "EDITOR" {
		return share.Permission == "EDITOR", nil
	}

	return false, nil
}

func (s *ShareService) ListSharedWithMeByUserID(ctx context.Context, userID string) ([]*model.Share, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.shareRepo.ListSharedWithMe(ctx, user.Email)
}
