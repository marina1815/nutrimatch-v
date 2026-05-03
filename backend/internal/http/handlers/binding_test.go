package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindStrictJSONRejectsUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"user@example.com","unknown":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var payload struct {
		Email string `json:"email"`
	}

	if err := bindStrictJSON(ctx, &payload); err == nil {
		t.Fatalf("expected unknown field rejection")
	}
}
