package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/services"
)

type RecommendationHandler struct {
	Service *services.RecommendationService
}

func (h *RecommendationHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	meals, err := h.Service.GetRecommendations(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "recommendations failed")
		return
	}

	c.JSON(http.StatusOK, dto.RecommendationResponse{
		ProfileID: c.Param("profileId"),
		Meals:     meals,
	})
}
