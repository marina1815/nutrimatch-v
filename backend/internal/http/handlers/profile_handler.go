package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/services"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type ProfileHandler struct {
	Profiles *services.ProfileService
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
	} `json:"lifestyle" validate:"required"`
	Preferences struct {
		Likes       []string `json:"likes"`
		Dislikes    []string `json:"dislikes"`
		MealStyles  []string `json:"mealStyles"`
		MealsPerDay int      `json:"mealsPerDay" validate:"required,gte=1,lte=8"`
	} `json:"preferences" validate:"required"`
	Constraints struct {
		Allergies           []string `json:"allergies"`
		Conditions          []string `json:"conditions"`
		ExcludedIngredients []string `json:"excludedIngredients"`
		HasChronicDisease   bool     `json:"hasChronicDisease"`
		ChronicDiseases     []string `json:"chronicDiseases"`
		TakesMedication     bool     `json:"takesMedication"`
		Medications         string   `json:"medications"`
	} `json:"constraints" validate:"required"`
}

func (h *ProfileHandler) Upsert(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid payload")
		return
	}

	if err := validation.Validate.Struct(req); err != nil {
		respondError(c, http.StatusBadRequest, "validation failed")
		return
	}

	userID := c.GetString("user_id")

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
	}
	preferences := &models.Preferences{
		Likes:       validation.NormalizeList(req.Preferences.Likes),
		Dislikes:    validation.NormalizeList(req.Preferences.Dislikes),
		MealStyles:  validation.NormalizeList(req.Preferences.MealStyles),
		MealsPerDay: req.Preferences.MealsPerDay,
	}
	constraints := &models.Constraints{
		Allergies:           validation.NormalizeList(req.Constraints.Allergies),
		Conditions:          validation.NormalizeList(req.Constraints.Conditions),
		ExcludedIngredients: validation.NormalizeList(req.Constraints.ExcludedIngredients),
		HasChronicDisease:   req.Constraints.HasChronicDisease,
		ChronicDiseases:     validation.NormalizeList(req.Constraints.ChronicDiseases),
		TakesMedication:     req.Constraints.TakesMedication,
		Medications:         validation.NormalizeString(req.Constraints.Medications),
	}

	if err := h.Profiles.Upsert(c.Request.Context(), userID, profile, lifestyle, preferences, constraints, validation.NormalizeString(req.Personal.FullName)); err != nil {
		respondError(c, http.StatusInternalServerError, "profile update failed")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ProfileHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	profile, lifestyle, preferences, constraints, fullName, err := h.Profiles.Get(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusNotFound, "profile not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"personal": gin.H{
			"fullName": fullName,
			"age":      profile.Age,
			"sex":      profile.Sex,
			"weight":   profile.Weight,
			"height":   profile.Height,
			"profession": profile.Profession,
			"city":     profile.City,
		},
		"lifestyle": gin.H{
			"activityLevel": lifestyle.ActivityLevel,
			"lifestyleType": lifestyle.LifestyleType,
			"goal":          lifestyle.Goal,
		},
		"preferences": gin.H{
			"likes":       preferences.Likes,
			"dislikes":    preferences.Dislikes,
			"mealStyles":  preferences.MealStyles,
			"mealsPerDay": preferences.MealsPerDay,
		},
		"constraints": gin.H{
			"allergies":           constraints.Allergies,
			"conditions":          constraints.Conditions,
			"excludedIngredients": constraints.ExcludedIngredients,
			"hasChronicDisease":   constraints.HasChronicDisease,
			"chronicDiseases":     constraints.ChronicDiseases,
			"takesMedication":     constraints.TakesMedication,
			"medications":         constraints.Medications,
		},
	})
}
