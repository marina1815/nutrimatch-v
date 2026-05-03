package catalog

import "testing"

func TestInferSafetyTagsCoversCoreMedicalRisks(t *testing.T) {
	tags := InferSafetyTags(
		"Creamy bacon cake",
		"Fried dessert with white flour and sugar",
		[]string{"bacon", "cream", "white flour", "sugar"},
		920,
		10,
		82,
		38,
		30,
		980,
	)

	for _, want := range []string{"diabetes-risk", "cholesterol-risk", "cardiac-risk", "hypertension-risk", "high-sodium", "saturated-fat"} {
		if !hasTag(tags, want) {
			t.Fatalf("expected tag %q in %v", want, tags)
		}
	}
}

func TestInferSafetyTagsCoversDigestiveAndMedicationSignals(t *testing.T) {
	tags := InferSafetyTags(
		"Spicy bean cabbage bowl",
		"Green chili with broccoli and spinach",
		[]string{"beans", "cabbage", "green chili", "broccoli", "spinach"},
		480,
		22,
		48,
		12,
		6,
		420,
	)

	for _, want := range []string{"digestive-risk", "gas-forming", "very-spicy", "vitamin-k-rich"} {
		if !hasTag(tags, want) {
			t.Fatalf("expected tag %q in %v", want, tags)
		}
	}
}

func TestInferSafetyTagsMarksDaouaraAsCholesterolRisk(t *testing.T) {
	tags := InferSafetyTags(
		"Chetitha Bouzelouf",
		"",
		[]string{"bouzelouf", "tomato", "onions", "spices"},
		480,
		28,
		45,
		14,
		5,
		520,
	)

	for _, want := range []string{"cholesterol-risk", "cardiac-risk", "red-meat"} {
		if !hasTag(tags, want) {
			t.Fatalf("expected tag %q in %v", want, tags)
		}
	}
}

func TestInferSafetyTagsMarksOnionRingsAsFriedRisk(t *testing.T) {
	tags := InferSafetyTags(
		"Onion Rings",
		"",
		[]string{"onions", "flour", "oil"},
		520,
		8,
		58,
		30,
		5,
		700,
	)

	for _, want := range []string{"fried", "cholesterol-risk", "cardiac-risk"} {
		if !hasTag(tags, want) {
			t.Fatalf("expected tag %q in %v", want, tags)
		}
	}
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
