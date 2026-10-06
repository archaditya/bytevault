package service

import (
	"context"
	"errors"

	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

type FolderService struct {
	repo         *repository.FolderRepository
	fileRepo     *repository.FileRepository
	activityRepo *repository.ActivityRepository
	storage      storage.StorageProvider
}

func (s *FolderService) SetStorage(storage storage.StorageProvider) {
	s.storage = storage
}

func NewFolderService(repo *repository.FolderRepository, fileRepo *repository.FileRepository, activityRepo *repository.ActivityRepository) *FolderService {
	return &FolderService{
		repo:         repo,
		fileRepo: fileRepo,
		activityRepo: activityRepo,
	}
}

func (s *FolderService) logActivity(ctx context.Context, userID, action, resourceID string, meta map[string]any) {
	if s.activityRepo == nil {
		return
	}

	resType := "folder"

	_ = s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: &resType,
		ResourceID:   &resourceID,
		Metadata:     meta,
	})
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
		IsPublic: false,
	}

	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}

	s.logActivity(ctx, userID, "folder.created", folder.ID, map[string]any{
		"folder_name": folder.Name,
		"parent_id":   parentID,
	})

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

	s.logActivity(ctx, userID, "folder.moved", id, map[string]any{
		"folder_name": folder.Name,
		"old_parent_id": folder.ParentID,
		"new_parent_id": parentID,
	})

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

	s.logActivity(ctx, userID, "folder.renamed", id, map[string]any{
		"folder_name": name,
		"old_name":    folder.Name,
	})

	return s.repo.Rename(ctx, id, name)
}

func (s *FolderService) ToggleShareStatus(ctx context.Context, id, userID string, isPublic bool) error {
	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil || folder.UserID != userID {
		return errors.New("folder not found or unauthorized")
	}

	s.logActivity(ctx, userID, "folder.shared", id, map[string]any{
		"folder_name": folder.Name,
		"is_public":   isPublic,
	})

	if err := s.repo.UpdatePublicStatus(ctx, id, isPublic); err != nil {
		return err
	}

	// Also sync all files currently inside this folder to match public status
	_ = s.fileRepo.UpdatePublicStatusByFolderID(ctx, id, isPublic)

	return nil
}

func (s *FolderService) GetPublicFolderContents(ctx context.Context, folderID string) (*model.Folder, []*model.Folder, []*model.File, []*model.Folder, error) {
	folder, err := s.repo.FindByIDPublic(ctx, folderID)
	if err != nil || folder == nil {
		return nil, nil, nil, nil, errors.New("public folder not found or not shared")
	}

	subfolders, err := s.repo.ListPublicSubfolders(ctx, folderID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	files, err := s.fileRepo.ListPublicFilesByFolderID(ctx, folderID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	breadcrumbs, _ := s.repo.GetPublicBreadcrumbs(ctx, folderID)

	// Increment access/views count asynchronously
	go func() {
		_ = s.repo.IncrementViews(context.Background(), folderID)
	}()

	// Enrich files with presigned thumbnail URLs if storage provider is present
	if s.storage != nil {
		for _, f := range files {
			if f.ThumbnailKey != nil && *f.ThumbnailKey != "" {
				if tURL, err := s.storage.GeneratePresignedDownloadURL(ctx, *f.ThumbnailKey, 1*time.Hour, "", true); err == nil {
					f.ThumbnailURL = &tURL
				}
			}
		}
	}

	return folder, subfolders, files, breadcrumbs, nil
}

func (s *FolderService) SavePublicFolderToVault(ctx context.Context, userID, publicFolderID string) (*model.Folder, int, error) {
	folder, _, files, _, err := s.GetPublicFolderContents(ctx, publicFolderID)
	if err != nil || folder == nil {
		return nil, 0, errors.New("public folder not found or not accessible")
	}

	// Create a new folder for the current user
	newFolder, err := s.CreateFolder(ctx, userID, folder.Name, nil)
	if err != nil {
		newFolder, err = s.CreateFolder(ctx, userID, folder.Name+" (Imported)", nil)
		if err != nil {
			return nil, 0, err
		}
	}

	count := 0
	for _, f := range files {
		newFile := &model.File{
			UserID:          userID,
			FolderID:        &newFolder.ID,
			Filename:        f.Filename,
			FileSize:        f.FileSize,
			ContentType:     f.ContentType,
			StorageKey:      f.StorageKey,
			StorageProvider: f.StorageProvider,
			ThumbnailKey:    f.ThumbnailKey,
			ContentHash:     f.ContentHash,
			Status:          "READY",
			IsPublic:        false,
			Tags:            f.Tags,
		}
		if err := s.fileRepo.Create(ctx, newFile); err == nil {
			count++
		}
	}

	s.logActivity(ctx, userID, "folder.import", newFolder.ID, map[string]any{
		"source_folder_id": publicFolderID,
		"imported_files":   count,
	})

	return newFolder, count, nil
}

func (s *FolderService) GetPublicFolderForZip(ctx context.Context, folderID string) (*model.Folder, []*model.File, storage.StorageProvider, error) {
	folder, err := s.repo.FindByIDPublic(ctx, folderID)
	if err != nil || folder == nil {
		return nil, nil, nil, errors.New("public folder not found or not shared")
	}

	files, err := s.fileRepo.ListPublicFilesByFolderID(ctx, folderID)
	if err != nil {
		return nil, nil, nil, err
	}

	if s.storage == nil {
		return nil, nil, nil, errors.New("storage provider not configured")
	}

	return folder, files, s.storage, nil
}

func (s *FolderService) DeleteFolder(ctx context.Context, id, userID string) error {
	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil || folder.UserID != userID {
		return errors.New("folder not found or unauthorized")
	}

	s.logActivity(ctx, userID, "folder.deleted", id, map[string]any{
		"folder_name": folder.Name,
	})

	return s.repo.SoftDelete(ctx, id)
}

func (s *FolderService) GetOrCreateFolder(ctx context.Context, userID, name string, parentID *string) (*model.Folder, error) {
	existing, err := s.repo.FindByName(ctx, userID, name, parentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return s.CreateFolder(ctx, userID, name, parentID)
}

