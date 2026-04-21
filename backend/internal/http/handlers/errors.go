package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/http/dto"
)

func respondOK[T any](c *gin.Context, status int, data T) {
	c.JSON(status, dto.SuccessResponse[T]{
		Data: data,
		Meta: responseMeta(c),
	})
}

func respondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.ErrorResponse{
		Error: dto.ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: responseMeta(c),
	})
}

func responseMeta(c *gin.Context) dto.ResponseMeta {
	return dto.ResponseMeta{
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().UTC(),
	}
}
