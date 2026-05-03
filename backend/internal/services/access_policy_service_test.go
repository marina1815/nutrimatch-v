package services

import "testing"

func TestAccessPolicyAllowsRecommendationChoiceForOwner(t *testing.T) {
	service := &AccessPolicyService{}
	decision := service.Decide(
		AccessSubject{UserID: "user-1", SessionID: "session-1", AuthMethod: "local"},
		"choose",
		AccessResource{OwnerUserID: "user-1", Sensitivity: "recommendation"},
	)
	if !decision.Allowed {
		t.Fatalf("expected owner to choose a recommendation, got %q", decision.Reason)
	}
}

func TestAccessPolicyRejectsRecommendationChoiceForOtherUser(t *testing.T) {
	service := &AccessPolicyService{}
	decision := service.Decide(
		AccessSubject{UserID: "user-1", SessionID: "session-1", AuthMethod: "local"},
		"choose",
		AccessResource{OwnerUserID: "user-2", Sensitivity: "recommendation"},
	)
	if decision.Allowed {
		t.Fatalf("expected cross-user recommendation choice to be denied")
	}
}
