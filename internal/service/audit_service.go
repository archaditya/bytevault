package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

type AuditService struct {
	repo *repository.AuditLogRepository
}

func NewAuditService(repo *repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

type LogInput struct {
	TenantID     *string
	ActorUserID  *string
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   *string
	Status       string
	RequestID    *string
	IPAddress    *string
	UserAgent    *string
	Metadata     map[string]interface{}
}

func (s *AuditService) Log(ctx context.Context, input LogInput) {
	if s == nil || s.repo == nil {
		return
	}

	var metaBytes json.RawMessage
	if input.Metadata != nil {
		b, err := json.Marshal(input.Metadata)
		if err == nil {
			metaBytes = b
		} else {
			metaBytes = []byte("{}")
		}
	} else {
		metaBytes = []byte("{}")
	}

	actorType := input.ActorType
	if actorType == "" {
		actorType = "user"
	}

	status := input.Status
	if status == "" {
		status = "success"
	}

	auditEntry := &model.AuditLog{
		TenantID:     input.TenantID,
		ActorUserID:  input.ActorUserID,
		ActorType:    actorType,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Status:       status,
		RequestID:    input.RequestID,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
		Metadata:     metaBytes,
	}

	// Execute asynchronously so business transactions are never blocked or delayed
	go func(entry *model.AuditLog) {
		bgCtx := context.Background()
		if err := s.repo.Create(bgCtx, entry); err != nil {
			log.Printf("[AUDIT ERROR] failed to write audit log: %v", err)
		}
	}(auditEntry)
}

func (s *AuditService) ListAuditLogs(ctx context.Context, filter model.AuditLogFilter) ([]model.AuditLog, int64, error) {
	return s.repo.List(ctx, filter)
}
