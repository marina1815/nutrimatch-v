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
	UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error
	UpdatePreferredMFAMethod(ctx context.Context, userID, method string) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, sessionID string) (*models.Session, error)
	GetByRefreshHash(ctx context.Context, refreshHash string) (*models.Session, error)
	Rotate(ctx context.Context, sessionID, oldRefreshHash, newRefreshHash, csrfBindingID string, expiresAt, idleExpiresAt time.Time) error
	Touch(ctx context.Context, sessionID string, idleExpiresAt time.Time) error
	Revoke(ctx context.Context, sessionID string) error
	RevokeForUser(ctx context.Context, userID, sessionID string) error
	RevokeOthers(ctx context.Context, userID, keepSessionID string) error
	ListByUser(ctx context.Context, userID string, limit int) ([]models.Session, error)
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

type DailyRecommendationRepository interface {
	GetActiveSet(ctx context.Context, userID, profileID string, now time.Time) (*models.DailyRecommendationSet, []*models.DailyRecommendationMeal, error)
	GetPreviousShownRecipeIDs(ctx context.Context, userID, profileID string, now time.Time) ([]string, error)
	GetSuppressedRecipeIDs(ctx context.Context, userID, profileID string, now time.Time) ([]string, error)
	CreateSet(ctx context.Context, set *models.DailyRecommendationSet, meals []*models.DailyRecommendationMeal) error
	ReplaceSetMeals(ctx context.Context, setID string, meals []*models.DailyRecommendationMeal, decisionSummary models.JSONMap, selectionMode, status string) error
	UpdateSetExplanations(ctx context.Context, setID string, explanations map[string]string, decisionSummary models.JSONMap, selectionMode string) error
	GetMealInActiveSet(ctx context.Context, userID, profileID, recipeID string, now time.Time) (*models.DailyRecommendationSet, *models.DailyRecommendationMeal, error)
	GetChoiceForSet(ctx context.Context, setID, userID, profileID string) (*models.RecipeChoice, error)
	CreateChoice(ctx context.Context, choice *models.RecipeChoice) error
}

type VectorRepository interface {
	UpsertProfileEmbedding(ctx context.Context, embedding *models.ProfileEmbedding) error
	UpsertRecipeEmbedding(ctx context.Context, embedding *models.RecipeEmbedding) error
	SearchSimilarProfileBundles(ctx context.Context, userID, profileID, embeddingVersion, vectorLiteral string, limit int) ([]ProfileBundle, error)
}

type MFARepository interface {
	GetTOTPSecret(ctx context.Context, userID string) (*models.TOTPSecret, error)
	UpsertTOTPSecret(ctx context.Context, secret *models.TOTPSecret) error
	EnableTOTP(ctx context.Context, userID string, confirmedAt time.Time) error
	DisableTOTP(ctx context.Context, userID string) error
	CreateWebAuthnCredential(ctx context.Context, credential *models.WebAuthnCredential) error
	ListWebAuthnCredentials(ctx context.Context, userID string) ([]models.WebAuthnCredential, error)
	UpdateWebAuthnCredentialUsed(ctx context.Context, credentialID string, usedAt time.Time) error
	CreateWebAuthnChallenge(ctx context.Context, challenge *models.WebAuthnChallenge) error
	ConsumeWebAuthnChallenge(ctx context.Context, userID, challengeID, kind string, now time.Time) (*models.WebAuthnChallenge, error)
	CreateMFALoginChallenge(ctx context.Context, challenge *models.MFALoginChallenge) error
	GetMFALoginChallenge(ctx context.Context, challengeID string, now time.Time) (*models.MFALoginChallenge, error)
	ConsumeMFALoginChallenge(ctx context.Context, challengeID string, now time.Time) (*models.MFALoginChallenge, error)
}

type AuditRepository interface {
	Create(ctx context.Context, event *models.AuditEvent) error
	AppendChained(ctx context.Context, event *models.AuditEvent, hash func(previousHash string, occurredAt time.Time) string) error
	LatestHash(ctx context.Context) (string, error)
	ListSince(ctx context.Context, since time.Time, limit int) ([]models.AuditEvent, error)
}

type AuthFailureRepository interface {
	Create(ctx context.Context, failure *models.AuthFailure) error
	CountRecent(ctx context.Context, emailHash, ipHash string, since time.Time) (int64, error)
}

type RateLimitBucketRepository interface {
	TakeToken(ctx context.Context, key, bucketType string, refillPerSecond float64, burst int, now time.Time) (bool, error)
}

type LocalRecipeRepository interface {
	Search(ctx context.Context, query LocalRecipeQuery) ([]LocalRecipeCandidate, error)
	SuggestIngredients(ctx context.Context, query string, limit int) ([]CatalogOption, error)
	ListAllergies(ctx context.Context) ([]CatalogOption, error)
}

type LocalRecipeQuery struct {
	QueryTerms          []string
	Likes               []string
	ExcludedIngredients []string
	AllergyKeys         []string
	ConditionKeys       []string
	Limit               int
}

type LocalRecipeCandidate struct {
	ID                   string
	Title                string
	Ingredients          []string
	Tags                 []string
	Calories             float64
	Protein              float64
	Carbs                float64
	Fat                  float64
	Sugar                float64
	SodiumMg             float64
	Score                float64
	MedicalCompatibility map[string]bool
	MedicalRiskFlags     []string
}

type CatalogOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Source string `json:"source,omitempty"`
}

type RetentionRepository interface {
	ApplyRetention(ctx context.Context, policy RetentionPolicy, now time.Time) error
}

type RetentionPolicy struct {
	AuthFailureRetentionDays         int
	RateLimitBucketRetentionHours    int
	SessionRetentionDays             int
	RecommendationTraceRetentionDays int
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
	Likes             []string
	Conditions        []string
	ChronicDiseases   []string
	HasChronicDisease bool
}

type Repositories struct {
	Users                UserRepository
	Profiles             ProfileRepository
	Sessions             SessionRepository
	MedicalRules         MedicalRuleRepository
	RecommendationRuns   RecommendationTraceRepository
	DailyRecommendations DailyRecommendationRepository
	Vectors              VectorRepository
	MFA                  MFARepository
	Audit                AuditRepository
	AuthFailures         AuthFailureRepository
	RateLimitBuckets     RateLimitBucketRepository
	ExternalIdentities   ExternalIdentityRepository
	LocalRecipes         LocalRecipeRepository
}

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
