package services

import (
	"context"
	"log"
	"time"

	"github.com/marina1815/nutrimatch/internal/repository"
)

type RetentionService struct {
	Repo   repository.RetentionRepository
	Policy repository.RetentionPolicy
}

func (s *RetentionService) Apply(ctx context.Context) error {
	if s == nil || s.Repo == nil {
		return nil
	}
	return s.Repo.ApplyRetention(ctx, s.Policy, time.Now().UTC())
}

func (s *RetentionService) Start(ctx context.Context, interval time.Duration, logger *log.Logger) {
	if s == nil || s.Repo == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Apply(ctx); err != nil && logger != nil {
					logger.Printf("retention cleanup failed: %v", err)
				}
			}
		}
	}()
}
