package taxonomy

import "testing"

func TestCanonicalizeIntoleranceFrenchAlias(t *testing.T) {
	got, ok := CanonicalizeIntolerance("Arachides")
	if !ok || got != "peanut" {
		t.Fatalf("expected peanut, got %q ok=%v", got, ok)
	}
}

func TestCanonicalizeIntoleranceShellfishAlias(t *testing.T) {
	got, ok := CanonicalizeIntolerance("Fruits de mer")
	if !ok || got != "shellfish" {
		t.Fatalf("expected shellfish, got %q ok=%v", got, ok)
	}
}

func TestCanonicalizeConditionFrenchAlias(t *testing.T) {
	got, ok := CanonicalizeCondition("Diabete")
	if !ok || got != "diabetes" {
		t.Fatalf("expected diabetes, got %q ok=%v", got, ok)
	}
}

func TestCanonicalizeMealStyleFrenchAlias(t *testing.T) {
	got, ok := CanonicalizeMealStyle("Equilibree")
	if !ok || got != "balanced" {
		t.Fatalf("expected balanced, got %q ok=%v", got, ok)
	}
}

func TestCanonicalizeIntoleranceTreeNut(t *testing.T) {
	got, ok := CanonicalizeIntolerance("noix")
	if !ok || got != "tree_nut" {
		t.Fatalf("expected tree_nut, got %q ok=%v", got, ok)
	}
}
