package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type fakeUserRepository struct {
	user           *models.User
	updateFullName string
}

func (r *fakeUserRepository) Create(_ context.Context, user *models.User) error {
	r.user = user
	if r.user.ID == "" {
		r.user.ID = "user-1"
	}
	return nil
}

func (r *fakeUserRepository) GetByEmail(_ context.Context, _ string) (*models.User, error) {
	if r.user == nil {
		return nil, errors.New("not found")
	}
	return r.user, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, _ string) (*models.User, error) {
	if r.user == nil {
		return nil, errors.New("not found")
	}
	return r.user, nil
}

func (r *fakeUserRepository) UpdateFullName(_ context.Context, _ string, fullName string) error {
	r.updateFullName = fullName
	return nil
}

type fakeProfileRepository struct {
	profile           *models.Profile
	lifestyle         *models.Lifestyle
	preferences       *models.Preferences
	constraints       *models.Constraints
	nutritionProfile  *models.NutritionProfile
	upsertProfile     bool
	upsertLifestyle   bool
	upsertPreferences bool
	upsertConstraints bool
	upsertNutrition   bool
}

func (r *fakeProfileRepository) UpsertProfile(_ context.Context, profile *models.Profile) error {
	r.upsertProfile = true
	r.profile = profile
	return nil
}

func (r *fakeProfileRepository) UpsertLifestyle(_ context.Context, lifestyle *models.Lifestyle) error {
	r.upsertLifestyle = true
	r.lifestyle = lifestyle
	return nil
}

func (r *fakeProfileRepository) UpsertPreferences(_ context.Context, preferences *models.Preferences) error {
	r.upsertPreferences = true
	r.preferences = preferences
	return nil
}

func (r *fakeProfileRepository) UpsertConstraints(_ context.Context, constraints *models.Constraints) error {
	r.upsertConstraints = true
	r.constraints = constraints
	return nil
}

func (r *fakeProfileRepository) GetProfile(_ context.Context, _ string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error) {
	if r.profile == nil || r.lifestyle == nil || r.preferences == nil || r.constraints == nil {
		return nil, nil, nil, nil, errors.New("not found")
	}
	return r.profile, r.lifestyle, r.preferences, r.constraints, nil
}

func (r *fakeProfileRepository) ListProfileBundles(_ context.Context, _ string, _ int) ([]repository.ProfileBundle, error) {
	return nil, nil
}

func (r *fakeProfileRepository) UpsertNutritionProfile(_ context.Context, profile *models.NutritionProfile) error {
	r.upsertNutrition = true
	r.nutritionProfile = profile
	return nil
}

func (r *fakeProfileRepository) GetNutritionProfile(_ context.Context, _ string) (*models.NutritionProfile, error) {
	if r.nutritionProfile == nil {
		return nil, errors.New("not found")
	}
	return r.nutritionProfile, nil
}

type fakeSessionRepository struct{}

func (r *fakeSessionRepository) Create(_ context.Context, _ *models.Session) error { return nil }
func (r *fakeSessionRepository) GetByID(_ context.Context, _ string) (*models.Session, error) {
	return nil, errors.New("not found")
}
func (r *fakeSessionRepository) GetByRefreshHash(_ context.Context, _ string) (*models.Session, error) {
	return nil, errors.New("not found")
}
func (r *fakeSessionRepository) Rotate(_ context.Context, _, _ string, _ time.Time) error { return nil }
func (r *fakeSessionRepository) Revoke(_ context.Context, _ string) error                 { return nil }

type fakeTxManager struct {
	called bool
	repos  repository.Repositories
}

func (m *fakeTxManager) WithinTransaction(_ context.Context, fn func(repository.Repositories) error) error {
	m.called = true
	return fn(m.repos)
}

type fakeRecipeSearcher struct {
	called bool
}

func (s *fakeRecipeSearcher) Search(_ context.Context, _ spoonacular.SearchOptions) (*spoonacular.SearchResponse, error) {
	s.called = true
	return &spoonacular.SearchResponse{}, nil
}

func TestBuildQuery(t *testing.T) {
	query := buildQuery([]string{"oriental"}, []string{"chicken"})
	if query == "" {
		t.Fatalf("expected query")
	}
}

func TestMatchMedicalRulesFindsMedicationAndConditionRules(t *testing.T) {
	rules := []models.MedicalRule{
		{Code: "diabetes_rule", ConditionKey: "diabetes"},
		{Code: "statin_rule", MedicationPattern: "statin"},
	}

	matched := MatchMedicalRules(rules, &models.Constraints{
		Conditions:  models.StringSlice{"diabetes"},
		Medications: "daily statin",
	})

	if len(matched) != 2 {
		t.Fatalf("expected two matched medical rules, got %d", len(matched))
	}
}

func TestProfileServiceUpsertUsesTransactionManager(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "Existing"}}
	profileRepo := &fakeProfileRepository{}
	txManager := &fakeTxManager{
		repos: repository.Repositories{
			Users:    userRepo,
			Profiles: profileRepo,
			Sessions: &fakeSessionRepository{},
		},
	}

	service := &ProfileService{
		Users:     userRepo,
		Profiles:  profileRepo,
		TxManager: txManager,
	}

	err := service.Upsert(
		context.Background(),
		"user-1",
		&models.Profile{},
		&models.Lifestyle{},
		&models.Preferences{},
		&models.Constraints{},
		"Updated User",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !txManager.called {
		t.Fatalf("expected transaction manager to be used")
	}
	if userRepo.updateFullName != "Updated User" {
		t.Fatalf("expected full name update inside transaction")
	}
	if !profileRepo.upsertProfile || !profileRepo.upsertLifestyle || !profileRepo.upsertPreferences || !profileRepo.upsertConstraints {
		t.Fatalf("expected all profile repositories to be updated inside transaction")
	}
}

func TestRecommendationServiceRejectsForeignProfileID(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "User"}}
	profileRepo := &fakeProfileRepository{
		profile:     &models.Profile{ID: "owned-profile", UserID: "user-1", Age: 25},
		lifestyle:   &models.Lifestyle{UserID: "user-1", Goal: "weight_loss", ActivityLevel: "light"},
		preferences: &models.Preferences{UserID: "user-1", MealsPerDay: 3},
		constraints: &models.Constraints{UserID: "user-1"},
	}
	searcher := &fakeRecipeSearcher{}
	service := &RecommendationService{
		Profiles: &ProfileService{Users: userRepo, Profiles: profileRepo},
		Recipes:  searcher,
	}

	_, err := service.GetRecommendations(context.Background(), "user-1", "other-profile", "req-1")
	if !errors.Is(err, ErrProfileAccessDenied) {
		t.Fatalf("expected ErrProfileAccessDenied, got %v", err)
	}
	if searcher.called {
		t.Fatalf("expected recipe search not to run for foreign profile IDs")
	}
}
