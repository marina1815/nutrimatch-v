package services

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type AuditService struct {
	Repo repository.AuditRepository
}

type AuditRecord struct {
	UserID        string
	SessionID     string
	EventType     string
	ResourceType  string
	ResourceID    string
	Outcome       string
	IP            string
	UserAgent     string
	RequestID     string
	Details       map[string]any
	ExternalTrace map[string]any
}

func (s *AuditService) Record(ctx context.Context, record AuditRecord) error {
	if s == nil || s.Repo == nil {
		return nil
	}

	return s.Repo.Create(ctx, &models.AuditEvent{
		UserID:        record.UserID,
		SessionID:     record.SessionID,
		EventType:     record.EventType,
		ResourceType:  record.ResourceType,
		ResourceID:    record.ResourceID,
		Outcome:       record.Outcome,
		IP:            record.IP,
		UserAgent:     record.UserAgent,
		RequestID:     record.RequestID,
		Details:       models.JSONMap(record.Details),
		ExternalTrace: models.JSONMap(record.ExternalTrace),
		OccurredAt:    time.Now(),
	})
}
