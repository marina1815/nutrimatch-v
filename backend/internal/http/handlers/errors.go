package handlers

import "github.com/gin-gonic/gin"

func respondError(c *gin.Context, status int, message string) {
	requestID := c.GetString("request_id")
	body := gin.H{"error": message}
	if requestID != "" {
		body["request_id"] = requestID
	}
	c.JSON(status, body)
}
