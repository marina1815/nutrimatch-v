package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/services"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type ProfileHandler struct {
	Profiles    *services.ProfileService
	Ingredients interface {
		Suggest(ctx context.Context, query string, limit int) ([]repository.CatalogOption, error)
		ListAllergies(ctx context.Context) ([]repository.CatalogOption, error)
	}
	Audit  *services.AuditService
	Access *services.AccessPolicyService
}

var profileConditionOptions = []repository.CatalogOption{
	{Value: "diabetes", Label: "Diabete", Source: "medical_rules"},
	{Value: "hypertension", Label: "Hypertension", Source: "medical_rules"},
	{Value: "cardiac", Label: "Maladie cardiaque", Source: "medical_rules"},
	{Value: "renal_failure", Label: "Insuffisance renale", Source: "medical_rules"},
	{Value: "hypercholesterolemia", Label: "Hypercholesterolemie", Source: "medical_rules"},
	{Value: "digestive_sensitivity", Label: "Sensibilite digestive", Source: "medical_rules"},
}

var profileChronicDiseaseOptions = []repository.CatalogOption{
	{Value: "diabetes", Label: "Diabete", Source: "medical_rules"},
	{Value: "hypertension", Label: "Hypertension", Source: "medical_rules"},
	{Value: "cardiac", Label: "Maladie cardiaque", Source: "medical_rules"},
	{Value: "renal_failure", Label: "Insuffisance renale", Source: "medical_rules"},
	{Value: "hypercholesterolemia", Label: "Hypercholesterolemie", Source: "medical_rules"},
	{Value: "digestive_sensitivity", Label: "Sensibilite digestive", Source: "medical_rules"},
}

type profileRequest struct {
	Personal struct {
		FullName string  `json:"fullName" validate:"required,min=2,max=120"`
		Age      int     `json:"age" validate:"required,gte=10,lte=120"`
		Sex      string  `json:"sex" validate:"required,oneof=male female"`
		Weight   float64 `json:"weight" validate:"required,gte=20,lte=400"`
		Height   float64 `json:"height" validate:"required,gte=80,lte=250"`
	} `json:"personal" validate:"required"`
	Lifestyle struct {
		ActivityLevel string `json:"activityLevel" validate:"required,oneof=sedentary light moderate active"`
		LifestyleType string `json:"lifestyleType" validate:"required,oneof=student employee athlete mixed other"`
		Goal          string `json:"goal" validate:"required,oneof=weight_loss muscle_gain weight_maintenance medical_diet energy_maintenance"`
	} `json:"lifestyle" validate:"required"`
	Preferences struct {
		Likes    []string `json:"likes" validate:"max=25,dive,min=1,max=50"`
		Dislikes []string `json:"dislikes" validate:"max=25,dive,min=1,max=50"`
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
	canonicalAllergies, allergiesSupported := h.canonicalizeAllowedAllergies(c.Request.Context(), req.Constraints.Allergies)
	if !allergiesSupported ||
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
		Age:    req.Personal.Age,
		Sex:    req.Personal.Sex,
		Weight: req.Personal.Weight,
		Height: req.Personal.Height,
	}
	lifestyle := &models.Lifestyle{
		ActivityLevel: req.Lifestyle.ActivityLevel,
		LifestyleType: req.Lifestyle.LifestyleType,
		Goal:          req.Lifestyle.Goal,
	}
	preferences := &models.Preferences{
		Likes:    validation.NormalizeList(req.Preferences.Likes),
		Dislikes: validation.NormalizeList(req.Preferences.Dislikes),
	}
	constraints := &models.Constraints{
		Allergies:           canonicalAllergies,
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
			"fullName": fullName,
			"age":      profile.Age,
			"sex":      profile.Sex,
			"weight":   profile.Weight,
			"height":   profile.Height,
		},
		"lifestyle": gin.H{
			"activityLevel": lifestyle.ActivityLevel,
			"lifestyleType": lifestyle.LifestyleType,
			"goal":          lifestyle.Goal,
		},
		"preferences": gin.H{
			"likes":    preferences.Likes,
			"dislikes": preferences.Dislikes,
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
		ProfileID:           nutritionProfile.ProfileID,
		BMI:                 nutritionProfile.BMI,
		BMICategory:         nutritionProfile.BMICategory,
		BMR:                 nutritionProfile.BMR,
		EstimatedCalories:   nutritionProfile.EstimatedCalories,
		TargetCalories:      nutritionProfile.TargetCalories,
		TargetProteinGrams:  nutritionProfile.TargetProteinGrams,
		TargetCarbsGrams:    nutritionProfile.TargetCarbsGrams,
		TargetFatGrams:      nutritionProfile.TargetFatGrams,
		MaxMealCalories:     nutritionProfile.MaxMealCalories,
		MinProteinPerMeal:   nutritionProfile.MinProteinPerMeal,
		MaxCarbsPerMeal:     nutritionProfile.MaxCarbsPerMeal,
		MaxFatPerMeal:       nutritionProfile.MaxFatPerMeal,
		MaxSugarPerMeal:     nutritionProfile.MaxSugarPerMeal,
		MaxSodiumMgPerMeal:  nutritionProfile.MaxSodiumMgPerMeal,
		DerivedRestrictions: []string(nutritionProfile.DerivedRestrictions),
		DerivedExcluded:     []string(nutritionProfile.DerivedExcluded),
		Metadata:            map[string]any(nutritionProfile.Metadata),
	})
}

func (h *ProfileHandler) SuggestIngredients(c *gin.Context) {
	if h.Ingredients == nil {
		respondOK(c, http.StatusOK, gin.H{"items": []repository.CatalogOption{}})
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

func (h *ProfileHandler) CatalogOptions(c *gin.Context) {
	userID := c.GetString("user_id")
	if !allowAccess(c, h.Access, "read", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_profile",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.taxonomy",
			ResourceType: "catalog.taxonomy",
			Outcome:      "denied",
		})
		return
	}

	allergies, err := h.listAllergyOptions(c.Request.Context())
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "profile.taxonomy",
			ResourceType: "catalog.taxonomy",
			Outcome:      "failed",
		})
		respondError(c, http.StatusBadGateway, "CATALOG_TAXONOMY_UNAVAILABLE", "catalog taxonomy unavailable")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "profile.taxonomy",
		ResourceType: "catalog.taxonomy",
		Outcome:      "success",
		Details:      map[string]any{"allergies": len(allergies), "conditions": len(profileConditionOptions)},
	})
	respondOK(c, http.StatusOK, gin.H{
		"allergies":       allergies,
		"conditions":      profileConditionOptions,
		"chronicDiseases": profileChronicDiseaseOptions,
	})
}

func (h *ProfileHandler) canonicalizeAllowedAllergies(ctx context.Context, values []string) ([]string, bool) {
	normalizedInput := validation.NormalizeList(values)
	if len(normalizedInput) == 0 {
		return []string{}, true
	}

	localAllowed, err := h.localAllergySet(ctx)
	if err != nil {
		return nil, false
	}

	out := make([]string, 0, len(normalizedInput))
	seen := make(map[string]struct{}, len(normalizedInput))
	for _, value := range normalizedInput {
		next := ""
		if canonical, ok := taxonomy.CanonicalizeIntolerance(value); ok {
			next = canonical
		} else {
			localKey := taxonomy.NormalizeLooseToken(value)
			if _, ok := localAllowed[localKey]; ok {
				next = localKey
			}
		}
		if next == "" {
			return nil, false
		}
		if _, exists := seen[next]; exists {
			return nil, false
		}
		seen[next] = struct{}{}
		out = append(out, next)
	}
	return out, true
}

func (h *ProfileHandler) localAllergySet(ctx context.Context) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if h.Ingredients == nil {
		return out, nil
	}
	options, err := h.Ingredients.ListAllergies(ctx)
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		key := taxonomy.NormalizeLooseToken(option.Value)
		if key != "" && allergyOptionVisible(option) {
			out[key] = struct{}{}
		}
	}
	return out, nil
}

func (h *ProfileHandler) listAllergyOptions(ctx context.Context) ([]repository.CatalogOption, error) {
	options := commonAllergyOptions()
	if h.Ingredients == nil {
		return normalizeVisibleAllergyOptions(options), nil
	}

	localOptions, err := h.Ingredients.ListAllergies(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(options)+len(localOptions))
	out := make([]repository.CatalogOption, 0, len(options)+len(localOptions))
	for _, option := range append(options, localOptions...) {
		if !allergyOptionVisible(option) {
			continue
		}
		value := taxonomy.NormalizeLooseToken(option.Value)
		label := strings.TrimSpace(option.Label)
		if value == "" || label == "" {
			continue
		}
		displayLabel := allergyDisplayLabel(value, label)
		if displayLabel == "" {
			continue
		}
		dedupeKey := allergyDedupeKey(value, displayLabel)
		if _, exists := seen[dedupeKey]; exists {
			continue
		}
		seen[dedupeKey] = struct{}{}
		out = append(out, repository.CatalogOption{
			Value:  value,
			Label:  displayLabel,
			Source: strings.TrimSpace(option.Source),
		})
	}
	return out, nil
}

func normalizeVisibleAllergyOptions(options []repository.CatalogOption) []repository.CatalogOption {
	out := make([]repository.CatalogOption, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		value := taxonomy.NormalizeLooseToken(option.Value)
		label := allergyDisplayLabel(value, option.Label)
		if value == "" || label == "" {
			continue
		}
		key := allergyDedupeKey(value, label)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, repository.CatalogOption{Value: value, Label: label, Source: option.Source})
	}
	return out
}

func commonAllergyOptions() []repository.CatalogOption {
	return []repository.CatalogOption{
		{Value: "peanut", Label: "Arachides", Source: "canonical"},
		{Value: "dairy", Label: "Lait", Source: "canonical"},
		{Value: "egg", Label: "Œufs", Source: "canonical"},
		{Value: "soy", Label: "Soja", Source: "canonical"},
		{Value: "seafood", Label: "Poisson", Source: "canonical"},
		{Value: "shellfish", Label: "Fruits de mer", Source: "canonical"},
		{Value: "gluten", Label: "Gluten", Source: "canonical"},
		{Value: "sesame", Label: "Sésame", Source: "canonical"},
		{Value: "tree_nut", Label: "Fruits à coque", Source: "canonical"},
		{Value: "wheat", Label: "Blé", Source: "canonical"},
	}
}

func allergyOptionVisible(option repository.CatalogOption) bool {
	value := taxonomy.NormalizeLooseToken(option.Value)
	label := taxonomy.NormalizeLooseToken(option.Label)
	source := taxonomy.NormalizeLooseToken(option.Source)
	combined := strings.Join([]string{value, label, source}, " ")
	switch {
	case value == "" || label == "":
		return false
	case strings.Contains(source, "croise"):
		return false
	case strings.Contains(combined, "cross between"):
		return false
	case strings.Contains(combined, "cross allergy"):
		return false
	case strings.Contains(combined, "pollen food") || strings.Contains(combined, "pfas"):
		return false
	case strings.Contains(combined, "oral") && strings.Contains(combined, "oas"):
		return false
	case strings.Contains(combined, "non allergy") || strings.Contains(combined, "not allergy"):
		return false
	case strings.HasPrefix(value, "cross ") || strings.HasPrefix(label, "cross "):
		return false
	default:
		return true
	}
}

func allergyDedupeKey(value, label string) string {
	if key := canonicalAllergyDedupeKey(value); key != "" {
		return key
	}
	normalized := allergyCanonicalLabelKey(label)
	switch taxonomy.NormalizeLooseToken(value) {
	case "egg", "egg allergy", "eggs", "oeuf", "oeufs", "œuf", "œufs":
		return "egg"
	case "dairy", "milk", "milk allergy", "lait":
		return "dairy"
	case "sesame", "sesame seed", "sesame allergy":
		return "sesame"
	case "wheat", "wheat allergy", "ble", "blé":
		return "wheat"
	case "tree nut", "tree_nut", "nut", "nuts":
		return "tree_nut"
	default:
		return normalized
	}
}

func canonicalAllergyDedupeKey(value string) string {
	switch taxonomy.NormalizeLooseToken(value) {
	case "egg", "egg allergy", "eggs", "oeuf", "oeufs":
		return "egg"
	case "dairy", "milk", "milk allergy", "lait":
		return "dairy"
	case "sesame", "sesame seed", "sesame allergy":
		return "sesame"
	case "wheat", "wheat allergy", "ble":
		return "wheat"
	case "tree nut", "tree_nut", "nut", "nuts":
		return "tree_nut"
	default:
		return ""
	}
}

func allergyCanonicalLabelKey(label string) string {
	key := taxonomy.NormalizeLooseToken(label)
	replacer := strings.NewReplacer(
		"œ", "oe",
		"é", "e",
		"è", "e",
		"ê", "e",
		"à", "a",
		"â", "a",
		"ù", "u",
		"û", "u",
		"ç", "c",
		"ï", "i",
		"î", "i",
	)
	key = replacer.Replace(key)
	key = strings.ReplaceAll(key, " allergy", "")
	key = strings.ReplaceAll(key, " intolerance", "")
	key = strings.Join(strings.Fields(key), " ")
	return key
}

func allergyDisplayLabel(value, fallback string) string {
	normalized := taxonomy.NormalizeLooseToken(value)
	if label, ok := canonicalAllergyLabel(normalized); ok {
		return label
	}
	switch normalized {
	case "milk allergy", "dairy":
		return "Lait"
	case "egg allergy", "egg":
		return "Œufs"
	case "peanut allergy", "peanut":
		return "Arachides"
	case "soy allergy", "soy":
		return "Soja"
	case "fish allergy", "seafood":
		return "Poisson"
	case "shellfish allergy", "shellfish":
		return "Fruits de mer"
	case "gluten allergy", "gluten":
		return "Gluten"
	case "sesame allergy", "sesame":
		return "Sésame"
	case "tree nut allergy", "tree nut", "tree_nut":
		return "Fruits à coque"
	case "wheat allergy", "wheat":
		return "Blé"
	default:
		cleaned := strings.TrimSpace(fallback)
		cleaned = strings.TrimSuffix(cleaned, " allergy")
		cleaned = strings.TrimSuffix(cleaned, " intolerance")
		cleaned = strings.TrimSuffix(cleaned, " syndrome")
		cleaned = strings.TrimSuffix(cleaned, " urticaria")
		if strings.EqualFold(cleaned, "non") || !allergyOptionVisible(repository.CatalogOption{Value: value, Label: fallback}) {
			return ""
		}
		return strings.TrimSpace(cleaned)
	}
}

func canonicalAllergyLabel(value string) (string, bool) {
	switch value {
	case "milk allergy", "dairy":
		return "Lait", true
	case "egg allergy", "egg":
		return "Oeufs", true
	case "peanut allergy", "peanut":
		return "Arachides", true
	case "soy allergy", "soy":
		return "Soja", true
	case "fish allergy", "seafood":
		return "Poisson", true
	case "shellfish allergy", "shellfish":
		return "Fruits de mer", true
	case "gluten allergy", "gluten":
		return "Gluten", true
	case "sesame allergy", "sesame":
		return "Sesame", true
	case "tree nut allergy", "tree nut", "tree_nut":
		return "Fruits a coque", true
	case "wheat allergy", "wheat":
		return "Ble", true
	default:
		return "", false
	}
}

func hasCanonicalMismatch(input []string, canonical []string) bool {
	return len(validation.NormalizeList(input)) != len(canonical)
}

func validateProfileConsistency(req profileRequest) string {
	likes := validation.NormalizeList(req.Preferences.Likes)
	dislikes := validation.NormalizeList(req.Preferences.Dislikes)
	excludedIngredients := validation.NormalizeList(req.Constraints.ExcludedIngredients)
	chronicDiseases := taxonomy.CanonicalizeConditionList(req.Constraints.ChronicDiseases)
	medications := validation.NormalizeString(req.Constraints.Medications)

	switch {
	case hasOverlap(likes, dislikes):
		return "likes_dislikes_overlap"
	case hasOverlap(likes, excludedIngredients):
		return "likes_excluded_overlap"
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
