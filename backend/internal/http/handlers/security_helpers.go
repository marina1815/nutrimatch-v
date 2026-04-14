package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/services"
)

func accessSubjectFromContext(c *gin.Context) services.AccessSubject {
	return services.AccessSubject{
		UserID:     c.GetString("user_id"),
		SessionID:  c.GetString("session_id"),
		AuthMethod: c.GetString("auth_method"),
	}
}

func allowAccess(c *gin.Context, policy *services.AccessPolicyService, action string, resource services.AccessResource) bool {
	if policy == nil {
		return true
	}
	if policy.Can(accessSubjectFromContext(c), action, resource) {
		return true
	}

	respondError(c, 403, "access denied")
	return false
}

func recordAudit(c *gin.Context, audit *services.AuditService, record services.AuditRecord) {
	if audit == nil {
		return
	}
	if record.UserID == "" {
		record.UserID = c.GetString("user_id")
	}
	if record.SessionID == "" {
		record.SessionID = c.GetString("session_id")
	}
	if record.RequestID == "" {
		record.RequestID = c.GetString("request_id")
	}
	if record.IP == "" {
		record.IP = c.ClientIP()
	}
	if record.UserAgent == "" {
		record.UserAgent = c.Request.UserAgent()
	}
	if record.Details == nil {
		record.Details = map[string]any{}
	}
	if record.ResourceType == "" {
		record.ResourceType = "application"
	}
	if record.Outcome == "" {
		record.Outcome = "success"
	}

	if authMethod := strings.TrimSpace(c.GetString("auth_method")); authMethod != "" {
		record.Details["authMethod"] = authMethod
	}
	_ = audit.Record(c.Request.Context(), record)
}
