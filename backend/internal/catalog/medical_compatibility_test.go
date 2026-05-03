package catalog

import "testing"

func TestAssessMedicalCompatibilityBlocksClosedConditionRisks(t *testing.T) {
	results := AssessMedicalCompatibility(
		"Sweet bacon fried rice",
		[]string{"white rice", "sugar", "bacon", "soy sauce", "onion"},
		900,
		42,
		82,
		32,
		24,
		980,
	)

	assertIncompatible(t, results, "diabetes")
	assertIncompatible(t, results, "hypertension")
	assertIncompatible(t, results, "cardiac")
	assertIncompatible(t, results, "renal_failure")
	assertIncompatible(t, results, "hypercholesterolemia")
	assertIncompatible(t, results, "digestive_sensitivity")
}

func TestAssessMedicalCompatibilityAllowsBalancedRecipe(t *testing.T) {
	results := AssessMedicalCompatibility(
		"Grilled chicken quinoa bowl",
		[]string{"chicken breast", "quinoa", "cucumber", "lettuce", "olive oil"},
		520,
		28,
		42,
		14,
		6,
		420,
	)

	for _, condition := range SupportedMedicalConditions {
		if !findCompatibility(results, condition).Compatible {
			t.Fatalf("expected %s to be compatible for balanced recipe, got %#v", condition, findCompatibility(results, condition))
		}
	}
}

func assertIncompatible(t *testing.T, results []MedicalCompatibility, condition string) {
	t.Helper()
	got := findCompatibility(results, condition)
	if got.ConditionKey == "" {
		t.Fatalf("missing compatibility result for %s", condition)
	}
	if got.Compatible {
		t.Fatalf("expected %s to be incompatible", condition)
	}
	if len(got.Reasons) == 0 {
		t.Fatalf("expected %s incompatibility reasons", condition)
	}
}

func findCompatibility(results []MedicalCompatibility, condition string) MedicalCompatibility {
	for _, result := range results {
		if result.ConditionKey == condition {
			return result
		}
	}
	return MedicalCompatibility{}
}
