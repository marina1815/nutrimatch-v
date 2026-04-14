package services

import "strings"

type AccessSubject struct {
	UserID     string
	SessionID  string
	AuthMethod string
}

type AccessResource struct {
	OwnerUserID string
	Sensitivity string
}

type AccessPolicyService struct{}

func (s *AccessPolicyService) Can(subject AccessSubject, action string, resource AccessResource) bool {
	if subject.UserID == "" || subject.SessionID == "" || resource.OwnerUserID == "" {
		return false
	}
	if subject.UserID != resource.OwnerUserID {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(subject.AuthMethod)) {
	case "local", "oidc":
	default:
		return false
	}

	switch strings.ToLower(strings.TrimSpace(resource.Sensitivity)) {
	case "health_profile", "nutrition_profile":
		return action == "read" || action == "write"
	case "recommendation":
		return action == "generate" || action == "read"
	case "health_trace":
		return action == "read" || action == "explain"
	default:
		return false
	}
}
