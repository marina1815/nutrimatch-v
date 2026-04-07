package services

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type UserLookup interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type UserLookupService struct {
	Users repository.UserRepository
}

func NewUserLookup(users repository.UserRepository) *UserLookupService {
	return &UserLookupService{Users: users}
}

func (s *UserLookupService) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.Users.GetByEmail(ctx, email)
}

