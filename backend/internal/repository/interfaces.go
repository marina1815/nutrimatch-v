package repository

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	UpdateFullName(ctx context.Context, userID, fullName string) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, sessionID string) (*models.Session, error)
	GetByRefreshHash(ctx context.Context, refreshHash string) (*models.Session, error)
	Rotate(ctx context.Context, sessionID, newRefreshHash string, expiresAt, idleExpiresAt time.Time) error
	Touch(ctx context.Context, sessionID string, idleExpiresAt time.Time) error
	Revoke(ctx context.Context, sessionID string) error
}

type ProfileRepository interface {
	UpsertProfile(ctx context.Context, profile *models.Profile) error
	UpsertLifestyle(ctx context.Context, lifestyle *models.Lifestyle) error
	UpsertPreferences(ctx context.Context, preferences *models.Preferences) error
	UpsertConstraints(ctx context.Context, constraints *models.Constraints) error
	GetProfile(ctx context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error)
	ListProfileBundles(ctx context.Context, excludeUserID string, limit int) ([]ProfileBundle, error)
	UpsertNutritionProfile(ctx context.Context, profile *models.NutritionProfile) error
	GetNutritionProfile(ctx context.Context, userID string) (*models.NutritionProfile, error)
}

type MedicalRuleRepository interface {
	ListActive(ctx context.Context) ([]models.MedicalRule, error)
}

type RecommendationTraceRepository interface {
	CreateRun(ctx context.Context, run *models.RecommendationRun) error
	ReplaceCandidates(ctx context.Context, runID string, candidates []*models.RecommendationCandidate) error
	GetLatestRunByProfile(ctx context.Context, userID, profileID string) (*models.RecommendationRun, []*models.RecommendationCandidate, error)
	GetCandidateByRecipeID(ctx context.Context, userID, profileID, recipeID string) (*models.RecommendationCandidate, error)
}

type AuditRepository interface {
	Create(ctx context.Context, event *models.AuditEvent) error
}

type AuthFailureRepository interface {
	Create(ctx context.Context, failure *models.AuthFailure) error
	CountRecent(ctx context.Context, emailHash, ipHash string, since time.Time) (int64, error)
}

type RateLimitBucketRepository interface {
	TakeToken(ctx context.Context, key, bucketType string, refillPerSecond float64, burst int, now time.Time) (bool, error)
}

type ExternalIdentityRepository interface {
	GetByProviderSubject(ctx context.Context, provider, issuer, subject string) (*models.ExternalIdentity, error)
	Create(ctx context.Context, identity *models.ExternalIdentity) error
	UpdateLogin(ctx context.Context, id string, email string, emailVerified bool, loginAt time.Time) error
}

type ProfileBundle struct {
	UserID            string
	Age               int
	ActivityLevel     string
	Goal              string
	MaxReadyTime      int
	MealStyles        []string
	MealTypes         []string
	PreferredCuisines []string
	Likes             []string
	Conditions        []string
	ChronicDiseases   []string
	HasChronicDisease bool
}

type Repositories struct {
	Users              UserRepository
	Profiles           ProfileRepository
	Sessions           SessionRepository
	MedicalRules       MedicalRuleRepository
	RecommendationRuns RecommendationTraceRepository
	Audit              AuditRepository
	AuthFailures       AuthFailureRepository
	RateLimitBuckets   RateLimitBucketRepository
	ExternalIdentities ExternalIdentityRepository
}

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
