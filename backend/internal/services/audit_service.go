package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type AuditService struct {
	Repo repository.AuditRepository
}

const zeroUUID = "00000000-0000-0000-0000-000000000000"

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

type AuditChainVerification struct {
	CheckedEvents int
	Valid         bool
	FirstFailure  string
}

func (s *AuditService) Record(ctx context.Context, record AuditRecord) error {
	if s == nil || s.Repo == nil {
		return nil
	}
	if record.UserID == "" {
		record.UserID = zeroUUID
	}
	if record.SessionID == "" {
		record.SessionID = zeroUUID
	}

	occurredAt := time.Now().UTC()
	previousHash, err := s.Repo.LatestHash(ctx)
	if err != nil {
		return err
	}
	eventHash := hashAuditRecord(previousHash, occurredAt, record)

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
		PreviousHash:  previousHash,
		EventHash:     eventHash,
		OccurredAt:    occurredAt,
	})
}

func (s *AuditService) VerifyChain(ctx context.Context, since time.Time, limit int) (*AuditChainVerification, error) {
	if s == nil || s.Repo == nil {
		return &AuditChainVerification{Valid: true}, nil
	}
	events, err := s.Repo.ListSince(ctx, since, limit)
	if err != nil {
		return nil, err
	}
	result := &AuditChainVerification{CheckedEvents: len(events), Valid: true}
	previousHash := ""
	if len(events) > 0 {
		previousHash = events[0].PreviousHash
	}
	for index, event := range events {
		record := AuditRecord{
			UserID:        event.UserID,
			SessionID:     event.SessionID,
			EventType:     event.EventType,
			ResourceType:  event.ResourceType,
			ResourceID:    event.ResourceID,
			Outcome:       event.Outcome,
			IP:            event.IP,
			UserAgent:     event.UserAgent,
			RequestID:     event.RequestID,
			Details:       map[string]any(event.Details),
			ExternalTrace: map[string]any(event.ExternalTrace),
		}
		if event.PreviousHash != previousHash {
			result.Valid = false
			result.FirstFailure = event.ID
			return result, nil
		}
		if hashAuditRecord(previousHash, event.OccurredAt.UTC(), record) != event.EventHash {
			result.Valid = false
			result.FirstFailure = event.ID
			return result, nil
		}
		previousHash = event.EventHash
		if index == len(events)-1 {
			result.Valid = true
		}
	}
	return result, nil
}

func hashAuditRecord(previousHash string, occurredAt time.Time, record AuditRecord) string {
	payload := struct {
		PreviousHash  string         `json:"previousHash"`
		OccurredAt    string         `json:"occurredAt"`
		UserID        string         `json:"userId"`
		SessionID     string         `json:"sessionId"`
		EventType     string         `json:"eventType"`
		ResourceType  string         `json:"resourceType"`
		ResourceID    string         `json:"resourceId"`
		Outcome       string         `json:"outcome"`
		IP            string         `json:"ip"`
		UserAgent     string         `json:"userAgent"`
		RequestID     string         `json:"requestId"`
		Details       map[string]any `json:"details"`
		ExternalTrace map[string]any `json:"externalTrace"`
	}{
		PreviousHash:  previousHash,
		OccurredAt:    occurredAt.Format(time.RFC3339Nano),
		UserID:        record.UserID,
		SessionID:     record.SessionID,
		EventType:     record.EventType,
		ResourceType:  record.ResourceType,
		ResourceID:    record.ResourceID,
		Outcome:       record.Outcome,
		IP:            record.IP,
		UserAgent:     record.UserAgent,
		RequestID:     record.RequestID,
		Details:       record.Details,
		ExternalTrace: record.ExternalTrace,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
