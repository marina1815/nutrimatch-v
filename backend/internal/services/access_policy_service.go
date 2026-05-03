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

type AccessDecision struct {
	Allowed bool
	Reason  string
}

func (s *AccessPolicyService) Can(subject AccessSubject, action string, resource AccessResource) bool {
	return s.Decide(subject, action, resource).Allowed
}

func (s *AccessPolicyService) Decide(subject AccessSubject, action string, resource AccessResource) AccessDecision {
	if subject.UserID == "" || subject.SessionID == "" || resource.OwnerUserID == "" {
		return deny("missing_subject_or_resource")
	}
	if subject.UserID != resource.OwnerUserID {
		return deny("owner_mismatch")
	}
	switch strings.ToLower(strings.TrimSpace(subject.AuthMethod)) {
	case "local", "oidc", "local+mfa:totp", "local+mfa:passkey":
	default:
		return deny("unsupported_auth_method")
	}

	action = strings.ToLower(strings.TrimSpace(action))
	switch strings.ToLower(strings.TrimSpace(resource.Sensitivity)) {
	case "health_profile", "nutrition_profile":
		return allowActions(action, "read", "write")
	case "recommendation":
		return allowActions(action, "generate", "read", "choose")
	case "health_trace":
		return allowActions(action, "read", "explain")
	case "identity_session":
		return allowActions(action, "read", "revoke")
	case "identity_mfa":
		return allowActions(action, "read", "enroll", "verify", "disable", "prefer")
	default:
		return deny("unknown_sensitivity")
	}
}

func allowActions(action string, allowed ...string) AccessDecision {
	for _, candidate := range allowed {
		if action == candidate {
			return AccessDecision{Allowed: true, Reason: "allowed"}
		}
	}
	return deny("action_not_allowed")
}

func deny(reason string) AccessDecision {
	return AccessDecision{Allowed: false, Reason: reason}
}
