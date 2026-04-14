package services

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
)

type ProfileService struct {
	Profiles     repository.ProfileRepository
	Users        repository.UserRepository
	TxManager    repository.TxManager
	Cipher       *security.Cipher
	Nutrition    *NutritionProfileService
	MedicalRules repository.MedicalRuleRepository
}

func (s *ProfileService) Upsert(ctx context.Context, userID string, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, fullName string) error {
	profile.UserID = userID
	lifestyle.UserID = userID
	preferences.UserID = userID
	constraints.UserID = userID

	if s.Cipher != nil {
		encrypted, err := s.Cipher.Encrypt(constraints.Medications)
		if err != nil {
			return err
		}
		constraints.Medications = encrypted
	}

	if s.TxManager == nil {
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
		if err := s.Profiles.UpsertConstraints(ctx, constraints); err != nil {
			return err
		}
		return s.recalculateNutrition(ctx, s.Profiles, profile, lifestyle, preferences, constraints)
	}

	return s.TxManager.WithinTransaction(ctx, func(repos repository.Repositories) error {
		if err := repos.Users.UpdateFullName(ctx, userID, fullName); err != nil {
			return err
		}
		if err := repos.Profiles.UpsertProfile(ctx, profile); err != nil {
			return err
		}
		if err := repos.Profiles.UpsertLifestyle(ctx, lifestyle); err != nil {
			return err
		}
		if err := repos.Profiles.UpsertPreferences(ctx, preferences); err != nil {
			return err
		}
		if err := repos.Profiles.UpsertConstraints(ctx, constraints); err != nil {
			return err
		}
		return s.recalculateNutrition(ctx, repos.Profiles, profile, lifestyle, preferences, constraints)
	})
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
	if s.Cipher != nil {
		decrypted, decryptErr := s.Cipher.Decrypt(constraints.Medications)
		if decryptErr != nil {
			return nil, nil, nil, nil, "", decryptErr
		}
		constraints.Medications = decrypted
	}
	return profile, lifestyle, preferences, constraints, user.FullName, nil
}

func (s *ProfileService) GetNutritionProfile(ctx context.Context, userID string) (*models.NutritionProfile, error) {
	profile, err := s.Profiles.GetNutritionProfile(ctx, userID)
	if err == nil {
		return profile, nil
	}
	if s.Nutrition == nil {
		return nil, err
	}
	return s.Nutrition.Recalculate(ctx, userID)
}

func (s *ProfileService) recalculateNutrition(ctx context.Context, repo repository.ProfileRepository, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints) error {
	if s.Nutrition == nil || s.MedicalRules == nil {
		return nil
	}

	plaintextConstraints := *constraints
	if s.Cipher != nil {
		decrypted, err := s.Cipher.Decrypt(constraints.Medications)
		if err != nil {
			return err
		}
		plaintextConstraints.Medications = decrypted
	}

	rules, err := s.MedicalRules.ListActive(ctx)
	if err != nil {
		return err
	}
	nutritionProfile := s.Nutrition.Build(profile, lifestyle, preferences, &plaintextConstraints, rules)
	return repo.UpsertNutritionProfile(ctx, nutritionProfile)
}
