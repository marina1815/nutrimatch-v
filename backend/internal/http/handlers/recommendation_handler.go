package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/services"
)

type RecommendationHandler struct {
	Service *services.RecommendationService
	Audit   *services.AuditService
	Access  *services.AccessPolicyService
}

func (h *RecommendationHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	profileID := c.Param("profileId")
	if !allowAccess(c, h.Access, "generate", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "recommendation",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.generate",
			ResourceType: "health.recommendation_run",
			ResourceID:   profileID,
			Outcome:      "denied",
		})
		return
	}

	response, err := h.Service.GetRecommendations(c.Request.Context(), userID, profileID, c.GetString("request_id"))
	if err != nil {
		if errors.Is(err, services.ErrProfileAccessDenied) {
			recordAudit(c, h.Audit, services.AuditRecord{
				UserID:       userID,
				EventType:    "recommendation.generate",
				ResourceType: "health.recommendation_run",
				ResourceID:   profileID,
				Outcome:      "denied",
				Details:      map[string]any{"reason": "profile_access_denied"},
			})
			respondError(c, http.StatusNotFound, "profile not found")
			return
		}
		if errors.Is(err, services.ErrRecommendationQuota) {
			recordAudit(c, h.Audit, services.AuditRecord{
				UserID:       userID,
				EventType:    "recommendation.generate",
				ResourceType: "health.recommendation_run",
				ResourceID:   profileID,
				Outcome:      "denied",
				Details:      map[string]any{"reason": "quota_exceeded"},
			})
			respondError(c, http.StatusTooManyRequests, "recommendation quota exceeded")
			return
		}
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.generate",
			ResourceType: "health.recommendation_run",
			ResourceID:   profileID,
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "recommendations failed")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "recommendation.generate",
		ResourceType: "health.recommendation_run",
		ResourceID:   response.RunID,
		Details: map[string]any{
			"profileId": profileID,
			"mealCount": len(response.Meals),
		},
	})
	c.JSON(http.StatusOK, response)
}

func (h *RecommendationHandler) Trace(c *gin.Context) {
	userID := c.GetString("user_id")
	profileID := c.Param("profileId")
	if !allowAccess(c, h.Access, "read", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_trace",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.trace",
			ResourceType: "health.recommendation_run",
			ResourceID:   profileID,
			Outcome:      "denied",
		})
		return
	}

	trace, err := h.Service.GetTrace(c.Request.Context(), userID, profileID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.trace",
			ResourceType: "health.recommendation_run",
			ResourceID:   profileID,
			Outcome:      "failed",
		})
		respondError(c, http.StatusNotFound, "recommendation trace not found")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "recommendation.trace",
		ResourceType: "health.recommendation_run",
		ResourceID:   trace.RunID,
		Details:      map[string]any{"profileId": profileID, "candidateCount": len(trace.Candidates)},
	})
	c.JSON(http.StatusOK, trace)
}

func (h *RecommendationHandler) Explain(c *gin.Context) {
	userID := c.GetString("user_id")
	profileID := c.Param("profileId")
	mealID := c.Query("mealId")
	if !allowAccess(c, h.Access, "explain", services.AccessResource{
		OwnerUserID: userID,
		Sensitivity: "health_trace",
	}) {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.explain",
			ResourceType: "health.recommendation_candidate",
			ResourceID:   mealID,
			Outcome:      "denied",
		})
		return
	}

	explanation, err := h.Service.GetExplanation(c.Request.Context(), userID, profileID, mealID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "recommendation.explain",
			ResourceType: "health.recommendation_candidate",
			ResourceID:   mealID,
			Outcome:      "failed",
		})
		respondError(c, http.StatusNotFound, "recommendation explanation not found")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "recommendation.explain",
		ResourceType: "health.recommendation_candidate",
		ResourceID:   mealID,
		Details:      map[string]any{"profileId": profileID, "runId": explanation.RunID},
	})
	c.JSON(http.StatusOK, explanation)
}
