package services

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
)

type NutritionProfileService struct {
	Profiles     repository.ProfileRepository
	MedicalRules repository.MedicalRuleRepository
	TxManager    repository.TxManager
}

func (s *NutritionProfileService) Build(profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, rules []models.MedicalRule) *models.NutritionProfile {
	heightMeters := profile.Height / 100.0
	bmi := round2(profile.Weight / (heightMeters * heightMeters))

	sexFactor := -161.0
	if strings.EqualFold(profile.Sex, "male") {
		sexFactor = 5
	}

	bmr := round2((10 * profile.Weight) + (6.25 * profile.Height) - (5 * float64(profile.Age)) + sexFactor)
	estimatedCalories := round2(bmr * activityMultiplier(lifestyle.ActivityLevel))
	targetCalories := round2(adjustCaloriesForGoal(estimatedCalories, lifestyle.Goal))

	proteinRatio, carbsRatio, fatRatio := macrosForGoal(lifestyle.Goal)
	targetProtein := round2((targetCalories * proteinRatio) / 4)
	targetCarbs := round2((targetCalories * carbsRatio) / 4)
	targetFat := round2((targetCalories * fatRatio) / 9)

	mealsPerDay := preferences.MealsPerDay
	if mealsPerDay <= 0 {
		mealsPerDay = 3
	}

	maxMealCalories := round2(targetCalories / float64(mealsPerDay) * 1.10)
	minProteinPerMeal := round2(targetProtein / float64(mealsPerDay) * 0.75)
	maxCarbsPerMeal := round2(targetCarbs / float64(mealsPerDay) * 1.10)
	maxFatPerMeal := round2(targetFat / float64(mealsPerDay) * 1.10)
	maxSugarPerMeal := 20.0
	maxSodiumMgPerMeal := 900.0

	derivedRestrictions := mergeStringSlices(preferences.Dislikes, constraints.Allergies, constraints.ExcludedIngredients)
	derivedExcluded := mergeStringSlices(constraints.Allergies, constraints.ExcludedIngredients)
	recommendedMealStyles := mergeStringSlices(preferences.MealStyles, goalMealStyles(lifestyle.Goal))

	matchedRules := MatchMedicalRules(rules, constraints)
	ruleCodes := make([]string, 0, len(matchedRules))
	for _, rule := range matchedRules {
		ruleCodes = append(ruleCodes, rule.Code)
		derivedExcluded = mergeStringSlices(derivedExcluded, rule.BlockedIngredients)
		recommendedMealStyles = mergeStringSlices(recommendedMealStyles, rule.RequiredTags)

		maxMealCalories = minPositive(maxMealCalories, rule.MaxCalories)
		minProteinPerMeal = maxPositive(minProteinPerMeal, rule.MinProteinGrams)
		maxCarbsPerMeal = minPositive(maxCarbsPerMeal, rule.MaxCarbsGrams)
		maxFatPerMeal = minPositive(maxFatPerMeal, rule.MaxFatGrams)
		maxSugarPerMeal = minPositive(maxSugarPerMeal, rule.MaxSugarGrams)
		maxSodiumMgPerMeal = minPositive(maxSodiumMgPerMeal, rule.MaxSodiumMg)
	}

	sort.Strings(ruleCodes)
	sort.Strings(recommendedMealStyles)
	sort.Strings(derivedExcluded)
	sort.Strings(derivedRestrictions)

	return &models.NutritionProfile{
		UserID:                profile.UserID,
		ProfileID:             profile.ID,
		BMI:                   bmi,
		BMICategory:           bmiCategory(bmi),
		BMR:                   bmr,
		EstimatedCalories:     estimatedCalories,
		TargetCalories:        targetCalories,
		TargetProteinGrams:    targetProtein,
		TargetCarbsGrams:      targetCarbs,
		TargetFatGrams:        targetFat,
		MaxMealCalories:       maxMealCalories,
		MinProteinPerMeal:     minProteinPerMeal,
		MaxCarbsPerMeal:       maxCarbsPerMeal,
		MaxFatPerMeal:         maxFatPerMeal,
		MaxSugarPerMeal:       maxSugarPerMeal,
		MaxSodiumMgPerMeal:    maxSodiumMgPerMeal,
		DerivedRestrictions:   models.StringSlice(derivedRestrictions),
		DerivedExcluded:       models.StringSlice(derivedExcluded),
		RecommendedMealStyles: models.StringSlice(recommendedMealStyles),
		Metadata: models.JSONMap{
			"matchedRuleCodes": ruleCodes,
			"goal":             lifestyle.Goal,
			"activityLevel":    lifestyle.ActivityLevel,
			"mealsPerDay":      mealsPerDay,
		},
		CalculatedAt: time.Now(),
	}
}

func (s *NutritionProfileService) Recalculate(ctx context.Context, userID string) (*models.NutritionProfile, error) {
	profile, lifestyle, preferences, constraints, err := s.Profiles.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	rules, err := s.MedicalRules.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	nutritionProfile := s.Build(profile, lifestyle, preferences, constraints, rules)
	if err := s.Profiles.UpsertNutritionProfile(ctx, nutritionProfile); err != nil {
		return nil, err
	}
	return nutritionProfile, nil
}

func activityMultiplier(level string) float64 {
	switch strings.ToLower(level) {
	case "active":
		return 1.725
	case "moderate":
		return 1.55
	case "light":
		return 1.375
	default:
		return 1.2
	}
}

func adjustCaloriesForGoal(calories float64, goal string) float64 {
	switch strings.ToLower(goal) {
	case "weight_loss":
		return calories - 450
	case "muscle_gain":
		return calories + 250
	case "medical_diet":
		return calories - 150
	default:
		return calories
	}
}

func macrosForGoal(goal string) (float64, float64, float64) {
	switch strings.ToLower(goal) {
	case "weight_loss":
		return 0.32, 0.33, 0.35
	case "muscle_gain":
		return 0.30, 0.45, 0.25
	case "medical_diet":
		return 0.28, 0.42, 0.30
	default:
		return 0.27, 0.43, 0.30
	}
}

func goalMealStyles(goal string) []string {
	switch strings.ToLower(goal) {
	case "weight_loss":
		return []string{"healthy", "balanced"}
	case "muscle_gain":
		return []string{"high-protein", "balanced"}
	case "medical_diet":
		return []string{"low-sodium", "balanced"}
	default:
		return []string{"balanced"}
	}
}

func mergeStringSlices(lists ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, list := range lists {
		for _, item := range list {
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
	}
	return out
}

func MatchMedicalRules(rules []models.MedicalRule, constraints *models.Constraints) []models.MedicalRule {
	conditions := mergeStringSlices(
		taxonomy.CanonicalizeConditionList([]string(constraints.Conditions)),
		taxonomy.CanonicalizeConditionList([]string(constraints.ChronicDiseases)),
	)
	medications := taxonomy.NormalizeLooseToken(constraints.Medications)
	out := make([]models.MedicalRule, 0)
	for _, rule := range rules {
		if rule.ConditionKey != "" {
			for _, condition := range conditions {
				if strings.EqualFold(condition, rule.ConditionKey) {
					out = append(out, rule)
					goto nextRule
				}
			}
		}
		if rule.MedicationPattern != "" && medications != "" && strings.Contains(medications, strings.ToLower(rule.MedicationPattern)) {
			out = append(out, rule)
		}
	nextRule:
	}
	return out
}

func bmiCategory(bmi float64) string {
	switch {
	case bmi < 18.5:
		return "underweight"
	case bmi < 25:
		return "normal"
	case bmi < 30:
		return "overweight"
	default:
		return "obese"
	}
}

func minPositive(current float64, value float64) float64 {
	if value <= 0 {
		return current
	}
	if current <= 0 || value < current {
		return value
	}
	return current
}

func maxPositive(current float64, value float64) float64 {
	if value <= 0 {
		return current
	}
	if value > current {
		return value
	}
	return current
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
