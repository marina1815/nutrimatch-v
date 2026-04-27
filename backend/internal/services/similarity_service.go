package services

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type SimilarityService struct {
	Profiles   repository.ProfileRepository
	Vectors    repository.VectorRepository
	Embeddings *EmbeddingService
	Semantic   SemanticExpander
}

type SimilaritySignals struct {
	Likes             []string
	MealStyles        []string
	MealTypes         []string
	Cuisines          []string
	Sources           []string
	DeterministicUsed bool
	SemanticUsed      bool
}

type SemanticExpander interface {
	Expand(ctx context.Context, userID string, existingLikes, existingMealStyles []string) (*SimilaritySignals, error)
}

type LocalSemanticExpander struct{}

var localSimilarityGraph = map[string][]string{
	"chicken": {"turkey", "lean chicken", "grilled chicken"},
	"rice":    {"brown rice", "quinoa", "bulgur"},
	"quinoa":  {"bulgur", "brown rice", "lentils"},
	"fish":    {"salmon", "tuna", "cod"},
	"salmon":  {"fish", "tuna"},
	"beans":   {"lentils", "chickpeas"},
	"lentils": {"beans", "chickpeas"},
}

var localStyleGraph = map[string][]string{
	"healthy":      {"balanced", "low-sugar"},
	"balanced":     {"healthy", "quick"},
	"high-protein": {"healthy", "balanced"},
	"low-sodium":   {"healthy", "balanced"},
	"quick":        {"balanced"},
}

func (e *LocalSemanticExpander) Expand(_ context.Context, _ string, existingLikes, existingMealStyles []string) (*SimilaritySignals, error) {
	likes := expandFromGraph(existingLikes, localSimilarityGraph, 8)
	styles := expandFromGraph(existingMealStyles, localStyleGraph, 4)
	return &SimilaritySignals{
		Likes:        dedupeLower(likes, existingLikes),
		MealStyles:   dedupeLower(styles, existingMealStyles),
		Sources:      []string{"local_semantic_graph"},
		SemanticUsed: len(likes) > 0 || len(styles) > 0,
	}, nil
}

func (s *SimilarityService) Expand(ctx context.Context, userID string, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints) (*SimilaritySignals, error) {
	if s == nil {
		return &SimilaritySignals{}, nil
	}

	signals := &SimilaritySignals{}
	existingLikes := []string(preferences.Likes)
	existingMealStyles := []string(preferences.MealStyles)
	deterministicSignals, err := s.expandFromPeerProfiles(ctx, userID, profile.Age, lifestyle.ActivityLevel, lifestyle.Goal, existingLikes, existingMealStyles)
	if err != nil {
		return nil, err
	}
	signals = mergeSimilaritySignals(signals, deterministicSignals)

	vectorSignals, err := s.expandFromVectorProfiles(ctx, userID, profile, lifestyle, preferences, constraints, existingLikes, existingMealStyles)
	if err != nil {
		return nil, err
	}
	signals = mergeSimilaritySignals(signals, vectorSignals)

	if s.Semantic != nil {
		semanticSignals, err := s.Semantic.Expand(ctx, userID, existingLikes, existingMealStyles)
		if err != nil {
			return nil, err
		}
		signals = mergeSimilaritySignals(signals, semanticSignals)
	}

	return signals, nil
}

func (s *SimilarityService) expandFromVectorProfiles(ctx context.Context, userID string, profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, existingLikes, existingMealStyles []string) (*SimilaritySignals, error) {
	if s == nil || s.Vectors == nil || s.Embeddings == nil || profile == nil {
		return &SimilaritySignals{}, nil
	}
	vectorLiteral, _, err := s.Embeddings.UpsertProfile(ctx, userID, profile, lifestyle, preferences, constraints)
	if err != nil {
		return nil, err
	}
	if vectorLiteral == "" {
		return &SimilaritySignals{}, nil
	}
	bundles, err := s.Vectors.SearchSimilarProfileBundles(ctx, userID, profile.ID, ProfileEmbeddingVersion, vectorLiteral, 10)
	if err != nil {
		return nil, err
	}

	likes := make([]string, 0)
	styles := make([]string, 0)
	mealTypes := make([]string, 0)
	cuisines := make([]string, 0)
	for _, bundle := range bundles {
		likes = append(likes, bundle.Likes...)
		styles = append(styles, bundle.MealStyles...)
		mealTypes = append(mealTypes, bundle.MealTypes...)
		cuisines = append(cuisines, bundle.PreferredCuisines...)
	}
	return &SimilaritySignals{
		Likes:             dedupeLower(likes, existingLikes),
		MealStyles:        dedupeLower(styles, existingMealStyles),
		MealTypes:         dedupeLower(mealTypes, nil),
		Cuisines:          dedupeLower(cuisines, nil),
		Sources:           []string{"pgvector_profile_embeddings"},
		DeterministicUsed: len(likes) > 0 || len(styles) > 0 || len(mealTypes) > 0 || len(cuisines) > 0,
	}, nil
}

func (s *SimilarityService) expandFromPeerProfiles(ctx context.Context, userID string, age int, activityLevel, goal string, existingLikes, existingMealStyles []string) (*SimilaritySignals, error) {
	if s == nil || s.Profiles == nil {
		return &SimilaritySignals{}, nil
	}

	bundles, err := s.Profiles.ListProfileBundles(ctx, userID, 25)
	if err != nil {
		return nil, err
	}

	type scored struct {
		bundle repository.ProfileBundle
		score  float64
	}
	scoredProfiles := make([]scored, 0, len(bundles))
	for _, bundle := range bundles {
		score := 0.0
		if strings.EqualFold(bundle.Goal, goal) {
			score += 3
		}
		if strings.EqualFold(bundle.ActivityLevel, activityLevel) {
			score += 2
		}
		score += math.Max(0, 2-(math.Abs(float64(bundle.Age-age))/10))
		score += overlapScore(existingLikes, bundle.Likes) * 1.5
		score += overlapScore(existingMealStyles, bundle.MealStyles)
		if bundle.HasChronicDisease {
			score -= 0.5
		}
		scoredProfiles = append(scoredProfiles, scored{bundle: bundle, score: score})
	}

	sort.SliceStable(scoredProfiles, func(i, j int) bool {
		return scoredProfiles[i].score > scoredProfiles[j].score
	})

	limit := 5
	if len(scoredProfiles) < limit {
		limit = len(scoredProfiles)
	}

	likes := make([]string, 0)
	styles := make([]string, 0)
	mealTypes := make([]string, 0)
	cuisines := make([]string, 0)
	for i := 0; i < limit; i++ {
		if scoredProfiles[i].score <= 0 {
			continue
		}
		likes = append(likes, scoredProfiles[i].bundle.Likes...)
		styles = append(styles, scoredProfiles[i].bundle.MealStyles...)
		mealTypes = append(mealTypes, scoredProfiles[i].bundle.MealTypes...)
		cuisines = append(cuisines, scoredProfiles[i].bundle.PreferredCuisines...)
	}

	return &SimilaritySignals{
		Likes:             dedupeLower(likes, existingLikes),
		MealStyles:        dedupeLower(styles, existingMealStyles),
		MealTypes:         dedupeLower(mealTypes, nil),
		Cuisines:          dedupeLower(cuisines, nil),
		Sources:           []string{"deterministic_peer_profiles"},
		DeterministicUsed: len(likes) > 0 || len(styles) > 0 || len(mealTypes) > 0 || len(cuisines) > 0,
	}, nil
}

func expandFromGraph(values []string, graph map[string][]string, limit int) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		for _, candidate := range graph[key] {
			normalized := strings.ToLower(strings.TrimSpace(candidate))
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
			if limit > 0 && len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func overlapScore(left, right []string) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(left))
	for _, item := range left {
		set[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}

	score := 0.0
	for _, item := range right {
		if _, ok := set[strings.ToLower(strings.TrimSpace(item))]; ok {
			score++
		}
	}
	return score
}

func dedupeLower(values []string, existing []string) []string {
	seen := make(map[string]struct{})
	for _, item := range existing {
		seen[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}

	out := make([]string, 0)
	for _, item := range values {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func mergeSimilaritySignals(base, incoming *SimilaritySignals) *SimilaritySignals {
	if base == nil {
		base = &SimilaritySignals{}
	}
	if incoming == nil {
		return base
	}

	return &SimilaritySignals{
		Likes:             mergeLists(base.Likes, incoming.Likes),
		MealStyles:        mergeLists(base.MealStyles, incoming.MealStyles),
		MealTypes:         mergeLists(base.MealTypes, incoming.MealTypes),
		Cuisines:          mergeLists(base.Cuisines, incoming.Cuisines),
		Sources:           mergeLists(base.Sources, incoming.Sources),
		DeterministicUsed: base.DeterministicUsed || incoming.DeterministicUsed,
		SemanticUsed:      base.SemanticUsed || incoming.SemanticUsed,
	}
}
