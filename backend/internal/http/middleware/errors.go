package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/http/dto"
)

func abortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error: dto.ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: dto.ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now().UTC(),
		},
	})
}
