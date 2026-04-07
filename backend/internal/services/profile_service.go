package services

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type ProfileService struct {
	Profiles repository.ProfileRepository
	Users    repository.UserRepository
}

func (s *ProfileService) Upsert(ctx context.Context, userID string, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, fullName string) error {
	profile.UserID = userID
	lifestyle.UserID = userID
	preferences.UserID = userID
	constraints.UserID = userID

	if err := s.Users.UpdateFullName(ctx, userID, fullName); err != nil {
		return err
	}
	if err := s.Profiles.UpsertProfile(ctx, profile); err != nil {
		return err
	}
	if err := s.Profiles.UpsertLifestyle(ctx, lifestyle); err != nil {
		return err
	}
	if err := s.Profiles.UpsertPreferences(ctx, preferences); err != nil {
		return err
	}
	return s.Profiles.UpsertConstraints(ctx, constraints)
}

func (s *ProfileService) Get(ctx context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, string, error) {
	profile, lifestyle, preferences, constraints, err := s.Profiles.GetProfile(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	return profile, lifestyle, preferences, constraints, user.FullName, nil
}
