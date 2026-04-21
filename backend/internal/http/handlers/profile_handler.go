package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/services"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type ProfileHandler struct {
	Profiles    *services.ProfileService
	Ingredients interface {
		Suggest(ctx context.Context, query string, limit int) ([]string, error)
	}
	Audit  *services.AuditService
	Access *services.AccessPolicyService
}

type profileRequest struct {
	Personal struct {
		FullName   string  `json:"fullName" validate:"required,min=2,max=120"`
		Age        int     `json:"age" validate:"required,gte=10,lte=120"`
		Sex        string  `json:"sex" validate:"required,oneof=male female"`
		Weight     float64 `json:"weight" validate:"required,gte=20,lte=400"`
		Height     float64 `json:"height" validate:"required,gte=80,lte=250"`
		Profession string  `json:"profession" validate:"required,max=120"`
		City       string  `json:"city" validate:"required,max=120"`
	} `json:"personal" validate:"required"`
	Lifestyle struct {
		ActivityLevel string `json:"activityLevel" validate:"required,oneof=sedentary light moderate active"`
		LifestyleType string `json:"lifestyleType" validate:"required,oneof=student employee athlete mixed other"`
		Goal          string `json:"goal" validate:"required,oneof=weight_loss muscle_gain weight_maintenance medical_diet energy_maintenance"`
		MaxReadyTime  int    `json:"maxReadyTime" validate:"required,gte=5,lte=240"`
	} `json:"lifestyle" validate:"required"`
	Preferences struct {
		Likes             []string `json:"likes" validate:"max=25,dive,min=1,max=50"`
		Dislikes          []string `json:"dislikes" validate:"max=25,dive,min=1,max=50"`
		MealStyles        []string `json:"mealStyles" validate:"max=20,dive,min=1,max=50"`
		MealTypes         []string `json:"mealTypes" validate:"max=6,dive,min=1,max=50"`
		PreferredCuisines []string `json:"preferredCuisines" validate:"max=8,dive,min=1,max=50"`
		ExcludedCuisines  []string `json:"excludedCuisines" validate:"max=8,dive,min=1,max=50"`
		MealsPerDay       int      `json:"mealsPerDay" validate:"required,gte=1,lte=8"`
	} `json:"preferences" validate:"required"`
	Constraints struct {
		Allergies           []string `json:"allergies" validate:"max=20,dive,min=1,max=50"`
		Conditions          []string `json:"conditions" validate:"max=20,dive,min=1,max=50"`
		ExcludedIngredients []string `json:"excludedIngredients" validate:"max=30,dive,min=1,max=50"`
		HasChronicDisease   bool     `json:"hasChronicDisease"`
		ChronicDiseases     []string `json:"chronicDiseases" validate:"max=10,dive,min=1,max=50"`
		TakesMedication     bool     `json:"takesMedication"`
		Medications         string   `json:"medications" validate:"max=250"`
	} `json:"constraints" validate:"required"`
}

const (
	maxFlexibleSignalCount = 40
	maxProfileTextBudget   = 1200
)

func (h *ProfileHandler) Upsert(c *gin.Context) {
	userID := c.GetString("user_id")
	if !allowAccess(c, h.Access, "write", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_profile",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
		})
		return
	}

	var req profileRequest
	if err := bindStrictJSON(c, &req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_payload"},
		})
		respondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	if err := validation.Validate.Struct(req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "validation_failed"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}
	if req.Constraints.HasChronicDisease && len(req.Constraints.ChronicDiseases) == 0 {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "missing_chronic_diseases"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}
	if req.Constraints.TakesMedication && validation.NormalizeString(req.Constraints.Medications) == "" {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "missing_medications"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}
	if reason := validateProfileConsistency(req); reason != "" {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": reason},
		})
		respondError(c, http.StatusBadRequest, "PROFILE_INCONSISTENT", "validation failed")
		return
	}
	if hasCanonicalMismatch(req.Preferences.MealStyles, taxonomy.CanonicalizeMealStyleList(req.Preferences.MealStyles)) ||
		hasCanonicalMismatch(req.Preferences.MealTypes, taxonomy.CanonicalizeMealTypeList(req.Preferences.MealTypes)) ||
		hasCanonicalMismatch(req.Preferences.PreferredCuisines, taxonomy.CanonicalizeCuisineList(req.Preferences.PreferredCuisines)) ||
		hasCanonicalMismatch(req.Preferences.ExcludedCuisines, taxonomy.CanonicalizeCuisineList(req.Preferences.ExcludedCuisines)) ||
		hasCanonicalMismatch(req.Constraints.Allergies, taxonomy.CanonicalizeIntoleranceList(req.Constraints.Allergies)) ||
		hasCanonicalMismatch(req.Constraints.Conditions, taxonomy.CanonicalizeConditionList(req.Constraints.Conditions)) ||
		hasCanonicalMismatch(req.Constraints.ChronicDiseases, taxonomy.CanonicalizeConditionList(req.Constraints.ChronicDiseases)) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "unsupported_canonical_value"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}

	profile := &models.Profile{
		Age:        req.Personal.Age,
		Sex:        req.Personal.Sex,
		Weight:     req.Personal.Weight,
		Height:     req.Personal.Height,
		Profession: validation.NormalizeString(req.Personal.Profession),
		City:       validation.NormalizeString(req.Personal.City),
	}
	lifestyle := &models.Lifestyle{
		ActivityLevel: req.Lifestyle.ActivityLevel,
		LifestyleType: req.Lifestyle.LifestyleType,
		Goal:          req.Lifestyle.Goal,
		MaxReadyTime:  req.Lifestyle.MaxReadyTime,
	}
	preferences := &models.Preferences{
		Likes:             validation.NormalizeList(req.Preferences.Likes),
		Dislikes:          validation.NormalizeList(req.Preferences.Dislikes),
		MealStyles:        taxonomy.CanonicalizeMealStyleList(req.Preferences.MealStyles),
		MealTypes:         taxonomy.CanonicalizeMealTypeList(req.Preferences.MealTypes),
		PreferredCuisines: taxonomy.CanonicalizeCuisineList(req.Preferences.PreferredCuisines),
		ExcludedCuisines:  taxonomy.CanonicalizeCuisineList(req.Preferences.ExcludedCuisines),
		MealsPerDay:       req.Preferences.MealsPerDay,
	}
	constraints := &models.Constraints{
		Allergies:           taxonomy.CanonicalizeIntoleranceList(req.Constraints.Allergies),
		Conditions:          taxonomy.CanonicalizeConditionList(req.Constraints.Conditions),
		ExcludedIngredients: validation.NormalizeList(req.Constraints.ExcludedIngredients),
		HasChronicDisease:   req.Constraints.HasChronicDisease,
		ChronicDiseases:     taxonomy.CanonicalizeConditionList(req.Constraints.ChronicDiseases),
		TakesMedication:     req.Constraints.TakesMedication,
		Medications:         validation.NormalizeString(req.Constraints.Medications),
	}

	if err := h.Profiles.Upsert(c.Request.Context(), userID, profile, lifestyle, preferences, constraints, validation.NormalizeString(req.Personal.FullName)); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "profile_update_failed"},
		})
		respondError(c, http.StatusInternalServerError, "PROFILE_UPDATE_FAILED", "profile update failed")
		return
	}

	savedProfile, _, _, _, _, err := h.Profiles.Get(c.Request.Context(), userID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.upsert",
			ResourceType: "health.profile",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "profile_readback_failed"},
		})
		respondError(c, http.StatusInternalServerError, "PROFILE_UPDATE_FAILED", "profile update failed")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "profile.upsert",
		ResourceType: "health.profile",
		ResourceID:   savedProfile.ID,
		Details: map[string]any{
			"goal":              lifestyle.Goal,
			"hasMedication":     constraints.TakesMedication,
			"hasChronicDisease": constraints.HasChronicDisease,
		},
	})
	respondOK(c, http.StatusOK, gin.H{"profileId": savedProfile.ID})
}

func (h *ProfileHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	includeSensitive := c.Query("includeSensitive") == "true"
	if !allowAccess(c, h.Access, "read", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_profile",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.read",
			ResourceType: "health.profile",
			Outcome:      "denied",
		})
		return
	}

	profile, lifestyle, preferences, constraints, fullName, err := h.Profiles.Get(c.Request.Context(), userID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.read",
			ResourceType: "health.profile",
			Outcome:      "failed",
		})
		respondError(c, http.StatusNotFound, "PROFILE_NOT_FOUND", "profile not found")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "profile.read",
		ResourceType: "health.profile",
		ResourceID:   profile.ID,
		Details: map[string]any{
			"includeSensitive": includeSensitive,
		},
	})

	medications := ""
	medicationsRedacted := false
	if includeSensitive {
		medications = constraints.Medications
	} else if constraints.TakesMedication && validation.NormalizeString(constraints.Medications) != "" {
		medicationsRedacted = true
	}
	respondOK(c, http.StatusOK, gin.H{
		"profileId": profile.ID,
		"personal": gin.H{
			"fullName":   fullName,
			"age":        profile.Age,
			"sex":        profile.Sex,
			"weight":     profile.Weight,
			"height":     profile.Height,
			"profession": profile.Profession,
			"city":       profile.City,
		},
		"lifestyle": gin.H{
			"activityLevel": lifestyle.ActivityLevel,
			"lifestyleType": lifestyle.LifestyleType,
			"goal":          lifestyle.Goal,
			"maxReadyTime":  lifestyle.MaxReadyTime,
		},
		"preferences": gin.H{
			"likes":             preferences.Likes,
			"dislikes":          preferences.Dislikes,
			"mealStyles":        preferences.MealStyles,
			"mealTypes":         preferences.MealTypes,
			"preferredCuisines": preferences.PreferredCuisines,
			"excludedCuisines":  preferences.ExcludedCuisines,
			"mealsPerDay":       preferences.MealsPerDay,
		},
		"constraints": gin.H{
			"allergies":           constraints.Allergies,
			"conditions":          constraints.Conditions,
			"excludedIngredients": constraints.ExcludedIngredients,
			"hasChronicDisease":   constraints.HasChronicDisease,
			"chronicDiseases":     constraints.ChronicDiseases,
			"takesMedication":     constraints.TakesMedication,
			"medications":         medications,
			"medicationsRedacted": medicationsRedacted,
		},
	})
}

func (h *ProfileHandler) GetNutrition(c *gin.Context) {
	userID := c.GetString("user_id")
	if !allowAccess(c, h.Access, "read", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "nutrition_profile",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "nutrition.read",
			ResourceType: "health.nutrition_profile",
			Outcome:      "denied",
		})
		return
	}

	nutritionProfile, err := h.Profiles.GetNutritionProfile(c.Request.Context(), userID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "nutrition.read",
			ResourceType: "health.nutrition_profile",
			Outcome:      "failed",
		})
		respondError(c, http.StatusNotFound, "NUTRITION_PROFILE_NOT_FOUND", "nutrition profile not found")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "nutrition.read",
		ResourceType: "health.nutrition_profile",
		ResourceID:   nutritionProfile.ID,
	})
	respondOK(c, http.StatusOK, dto.NutritionProfileResponse{
		ProfileID:             nutritionProfile.ProfileID,
		BMI:                   nutritionProfile.BMI,
		BMICategory:           nutritionProfile.BMICategory,
		BMR:                   nutritionProfile.BMR,
		EstimatedCalories:     nutritionProfile.EstimatedCalories,
		TargetCalories:        nutritionProfile.TargetCalories,
		TargetProteinGrams:    nutritionProfile.TargetProteinGrams,
		TargetCarbsGrams:      nutritionProfile.TargetCarbsGrams,
		TargetFatGrams:        nutritionProfile.TargetFatGrams,
		MaxMealCalories:       nutritionProfile.MaxMealCalories,
		MinProteinPerMeal:     nutritionProfile.MinProteinPerMeal,
		MaxCarbsPerMeal:       nutritionProfile.MaxCarbsPerMeal,
		MaxFatPerMeal:         nutritionProfile.MaxFatPerMeal,
		MaxSugarPerMeal:       nutritionProfile.MaxSugarPerMeal,
		MaxSodiumMgPerMeal:    nutritionProfile.MaxSodiumMgPerMeal,
		DerivedRestrictions:   []string(nutritionProfile.DerivedRestrictions),
		DerivedExcluded:       []string(nutritionProfile.DerivedExcluded),
		RecommendedMealStyles: []string(nutritionProfile.RecommendedMealStyles),
		Metadata:              map[string]any(nutritionProfile.Metadata),
	})
}

func (h *ProfileHandler) SuggestIngredients(c *gin.Context) {
	if h.Ingredients == nil {
		respondOK(c, http.StatusOK, gin.H{"items": []string{}})
		return
	}

	userID := c.GetString("user_id")
	if !allowAccess(c, h.Access, "read", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_profile",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "ingredient.suggest",
			ResourceType: "catalog.ingredient",
			Outcome:      "denied",
		})
		return
	}

	query := validation.NormalizeString(c.Query("q"))
	limit := 5
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 1 && parsed <= 10 {
			limit = parsed
		}
	}

	items, err := h.Ingredients.Suggest(c.Request.Context(), query, limit)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "ingredient.suggest",
			ResourceType: "catalog.ingredient",
			Outcome:      "failed",
		})
		respondError(c, http.StatusBadGateway, "INGREDIENT_SUGGESTION_UNAVAILABLE", "ingredient suggestion unavailable")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "ingredient.suggest",
		ResourceType: "catalog.ingredient",
		Outcome:      "success",
		Details: map[string]any{
			"queryLength": len(query),
			"resultCount": len(items),
		},
	})
	respondOK(c, http.StatusOK, gin.H{"items": items})
}

func hasCanonicalMismatch(input []string, canonical []string) bool {
	return len(validation.NormalizeList(input)) != len(canonical)
}

func validateProfileConsistency(req profileRequest) string {
	likes := validation.NormalizeList(req.Preferences.Likes)
	dislikes := validation.NormalizeList(req.Preferences.Dislikes)
	excludedIngredients := validation.NormalizeList(req.Constraints.ExcludedIngredients)
	preferredCuisines := taxonomy.CanonicalizeCuisineList(req.Preferences.PreferredCuisines)
	excludedCuisines := taxonomy.CanonicalizeCuisineList(req.Preferences.ExcludedCuisines)
	chronicDiseases := taxonomy.CanonicalizeConditionList(req.Constraints.ChronicDiseases)
	medications := validation.NormalizeString(req.Constraints.Medications)

	switch {
	case hasOverlap(likes, dislikes):
		return "likes_dislikes_overlap"
	case hasOverlap(likes, excludedIngredients):
		return "likes_excluded_overlap"
	case hasOverlap(preferredCuisines, excludedCuisines):
		return "preferred_excluded_cuisines_overlap"
	case !req.Constraints.HasChronicDisease && len(chronicDiseases) > 0:
		return "unexpected_chronic_diseases"
	case !req.Constraints.TakesMedication && medications != "":
		return "unexpected_medications"
	}

	flexibleSignalCount := len(likes) + len(dislikes) + len(excludedIngredients)
	if flexibleSignalCount > maxFlexibleSignalCount {
		return "payload_too_complex"
	}

	textBudget := len(validation.NormalizeString(req.Personal.FullName)) +
		len(validation.NormalizeString(req.Personal.Profession)) +
		len(validation.NormalizeString(req.Personal.City)) +
		len(medications)
	for _, item := range likes {
		textBudget += len(item)
	}
	for _, item := range dislikes {
		textBudget += len(item)
	}
	for _, item := range excludedIngredients {
		textBudget += len(item)
	}
	if textBudget > maxProfileTextBudget {
		return "payload_too_large"
	}

	return ""
}

func hasOverlap(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}

	lookup := make(map[string]struct{}, len(left))
	for _, item := range left {
		lookup[item] = struct{}{}
	}
	for _, item := range right {
		if _, exists := lookup[item]; exists {
			return true
		}
	}
	return false
}
