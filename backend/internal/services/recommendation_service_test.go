package services

import "testing"

func TestBuildQuery(t *testing.T) {
	query := buildQuery([]string{"oriental"}, []string{"chicken"})
	if query == "" {
		t.Fatalf("expected query")
	}
}

