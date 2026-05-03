package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func (h *HealthHandler) Ping(c *gin.Context) {
	respondOK(c, http.StatusOK, gin.H{"status": "ok"})
}
