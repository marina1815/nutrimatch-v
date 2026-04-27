package services

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
)

const ProfileEmbeddingVersion = "profile-hash-v1-768"
const embeddingDimensions = 768

type EmbeddingService struct {
	Vectors repository.VectorRepository
}

type profileEmbeddingPayload struct {
	Age             int      `json:"age"`
	ActivityLevel   string   `json:"activityLevel"`
	Goal            string   `json:"goal"`
	Likes           []string `json:"likes"`
	Dislikes        []string `json:"dislikes"`
	MealStyles      []string `json:"mealStyles"`
	MealTypes       []string `json:"mealTypes"`
	Cuisines        []string `json:"cuisines"`
	Allergies       []string `json:"allergies"`
	Conditions      []string `json:"conditions"`
	ChronicDiseases []string `json:"chronicDiseases"`
	MedicationFlag  bool     `json:"medicationFlag"`
}

func (s *EmbeddingService) UpsertProfile(ctx context.Context, userID string, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints) (string, string, error) {
	if s == nil || s.Vectors == nil || profile == nil || lifestyle == nil || preferences == nil || constraints == nil {
		return "", "", nil
	}
	payload := buildProfileEmbeddingPayload(profile, lifestyle, preferences, constraints)
	vectorLiteral, sourceHash := vectorizePayload(payload)
	err := s.Vectors.UpsertProfileEmbedding(ctx, &models.ProfileEmbedding{
		UserID:           userID,
		ProfileID:        profile.ID,
		EmbeddingVersion: ProfileEmbeddingVersion,
		SourceHash:       sourceHash,
		Embedding:        vectorLiteral,
		Metadata: models.JSONMap{
			"source": "deterministic_profile_vectorizer",
		},
	})
	return vectorLiteral, sourceHash, err
}

func buildProfileEmbeddingPayload(profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints) profileEmbeddingPayload {
	return profileEmbeddingPayload{
		Age:             profile.Age,
		ActivityLevel:   taxonomy.NormalizeLooseToken(lifestyle.ActivityLevel),
		Goal:            taxonomy.NormalizeLooseToken(lifestyle.Goal),
		Likes:           sortedNormalized([]string(preferences.Likes)),
		Dislikes:        sortedNormalized([]string(preferences.Dislikes)),
		MealStyles:      sortedNormalized([]string(preferences.MealStyles)),
		MealTypes:       sortedNormalized([]string(preferences.MealTypes)),
		Cuisines:        sortedNormalized(append([]string(preferences.PreferredCuisines), []string(preferences.ExcludedCuisines)...)),
		Allergies:       sortedNormalized([]string(constraints.Allergies)),
		Conditions:      sortedNormalized([]string(constraints.Conditions)),
		ChronicDiseases: sortedNormalized([]string(constraints.ChronicDiseases)),
		MedicationFlag:  constraints.TakesMedication,
	}
}

func sortedNormalized(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key := taxonomy.NormalizeLooseToken(value)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func vectorizePayload(payload profileEmbeddingPayload) (string, string) {
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	sourceHash := hex.EncodeToString(sum[:])

	weights := make([]float64, embeddingDimensions)
	addTokens(weights, "age:"+strconv.Itoa(payload.Age/10), 0.6)
	addTokens(weights, "activity:"+payload.ActivityLevel, 2.0)
	addTokens(weights, "goal:"+payload.Goal, 2.2)
	addTokenList(weights, payload.Likes, "like", 1.6)
	addTokenList(weights, payload.Dislikes, "dislike", -0.7)
	addTokenList(weights, payload.MealStyles, "style", 1.4)
	addTokenList(weights, payload.MealTypes, "meal_type", 1.2)
	addTokenList(weights, payload.Cuisines, "cuisine", 0.9)
	addTokenList(weights, payload.Allergies, "allergy", -2.0)
	addTokenList(weights, payload.Conditions, "condition", -1.8)
	addTokenList(weights, payload.ChronicDiseases, "chronic", -2.0)
	if payload.MedicationFlag {
		addTokens(weights, "medication:true", -1.4)
	}

	norm := 0.0
	for _, value := range weights {
		norm += value * value
	}
	if norm > 0 {
		scale := math.Sqrt(norm)
		for i := range weights {
			weights[i] = weights[i] / scale
		}
	}

	var b strings.Builder
	b.WriteString("[")
	for i, value := range weights {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%.8f", value))
	}
	b.WriteString("]")
	return b.String(), sourceHash
}

func addTokenList(weights []float64, values []string, namespace string, weight float64) {
	for _, value := range values {
		addTokens(weights, namespace+":"+value, weight)
	}
}

func addTokens(weights []float64, token string, weight float64) {
	if strings.TrimSpace(token) == "" {
		return
	}
	sum := sha256.Sum256([]byte(token))
	index := binary.BigEndian.Uint64(sum[:8]) % uint64(len(weights))
	sign := 1.0
	if sum[8]&1 == 1 {
		sign = -1
	}
	weights[index] += weight * sign
}
