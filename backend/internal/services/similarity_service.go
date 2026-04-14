package services

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/marina1815/nutrimatch/internal/repository"
)

type SimilarityService struct {
	Profiles repository.ProfileRepository
}

type SimilaritySignals struct {
	Likes      []string
	MealStyles []string
}

func (s *SimilarityService) Expand(ctx context.Context, userID string, age int, activityLevel, goal string, existingLikes, existingMealStyles []string) (*SimilaritySignals, error) {
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
	for i := 0; i < limit; i++ {
		if scoredProfiles[i].score <= 0 {
			continue
		}
		likes = append(likes, scoredProfiles[i].bundle.Likes...)
		styles = append(styles, scoredProfiles[i].bundle.MealStyles...)
	}

	return &SimilaritySignals{
		Likes:      dedupeLower(likes, existingLikes),
		MealStyles: dedupeLower(styles, existingMealStyles),
	}, nil
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
