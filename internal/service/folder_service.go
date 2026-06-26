package service

import (
	"context"
	"errors"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

type FolderService struct {
	repo     *repository.FolderRepository
	fileRepo *repository.FileRepository
}

func NewFolderService(repo *repository.FolderRepository, fileRepo *repository.FileRepository) *FolderService {
	return &FolderService{
		repo:     repo,
		fileRepo: fileRepo,
	}
}

func (s *FolderService) CreateFolder(ctx context.Context, userID, name string, parentID *string) (*model.Folder, error) {
	if name == "" {
		return nil, errors.New("folder name cannot be empty")
	}

	// If parent ID is provided, verify it exists and belongs to the user
	if parentID != nil && *parentID != "" {
		parent, err := s.repo.FindByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.UserID != userID {
			return nil, errors.New("parent folder not found or unauthorized")
		}
	} else {
		parentID = nil
	}

	folder := &model.Folder{
		UserID:   userID,
		Name:     name,
		ParentID: parentID,
	}

	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

func (s *FolderService) ListFolders(ctx context.Context, userID string, parentID *string, flat bool) ([]*model.Folder, error) {
	if flat {
		return s.repo.ListAllFlat(ctx, userID)
	}
	return s.repo.ListByUserID(ctx, userID, parentID)
}

func (s *FolderService) MoveFolder(ctx context.Context, id, userID string, parentID *string) error {
	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil || folder.UserID != userID {
		return errors.New("folder not found or unauthorized")
	}

	if parentID != nil && *parentID == "" {
		parentID = nil
	}

	// Prevent cyclic nesting (cannot move a folder inside itself)
	if parentID != nil && *parentID == id {
		return errors.New("cannot move a folder inside itself")
	}

	// Verify target parent exists and belongs to the user
	if parentID != nil {
		parent, err := s.repo.FindByID(ctx, *parentID)
		if err != nil {
			return err
		}
		if parent == nil || parent.UserID != userID {
			return errors.New("target folder not found or unauthorized")
		}
	}

	return s.repo.UpdateParent(ctx, id, parentID)
}

func (s *FolderService) RenameFolder(ctx context.Context, id, userID, name string) error {
	if name == "" {
		return errors.New("folder name cannot be empty")
	}

	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil || folder.UserID != userID {
		return errors.New("folder not found or unauthorized")
	}

	return s.repo.Rename(ctx, id, name)
}

func (s *FolderService) DeleteFolder(ctx context.Context, id, userID string) error {
	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil || folder.UserID != userID {
		return errors.New("folder not found or unauthorized")
	}

	return s.repo.SoftDelete(ctx, id)
}
