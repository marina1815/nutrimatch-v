package catalog

import (
	"sort"
	"strings"
)

const MedicalCompatibilitySource = "deterministic_medical_matrix_v1"

var SupportedMedicalConditions = []string{
	"diabetes",
	"hypertension",
	"cardiac",
	"renal_failure",
	"hypercholesterolemia",
	"digestive_sensitivity",
}

type MedicalCompatibility struct {
	ConditionKey string
	Compatible   bool
	Reasons      []string
	RiskFlags    []string
}

// AssessMedicalCompatibility applies the closed NutriMatch medical rule matrix
// to one recipe. It is intentionally deterministic and independent from AI:
// selected conditions can only narrow the safe recipe pool, never loosen it.
func AssessMedicalCompatibility(title string, ingredients []string, calories, protein, carbs, fat, sugar, sodium float64) []MedicalCompatibility {
	text := normalizeTagText(title + " " + strings.Join(ingredients, " "))
	out := make([]MedicalCompatibility, 0, len(SupportedMedicalConditions))

	hasAny := func(patterns ...string) bool {
		for _, pattern := range patterns {
			if containsTagTerm(text, pattern) {
				return true
			}
		}
		return false
	}
	appendResult := func(condition string, reasons, flags []string) {
		out = append(out, MedicalCompatibility{
			ConditionKey: condition,
			Compatible:   len(reasons) == 0,
			Reasons:      sortedUnique(reasons),
			RiskFlags:    sortedUnique(flags),
		})
	}

	diabetesReasons := []string{}
	diabetesFlags := []string{}
	if sugar >= 16 {
		diabetesReasons = append(diabetesReasons, "sugar_over_limit")
		diabetesFlags = append(diabetesFlags, "high_sugar")
	}
	if carbs >= 65 {
		diabetesReasons = append(diabetesReasons, "carbs_over_limit")
		diabetesFlags = append(diabetesFlags, "high_carb")
	}
	if hasAny(
		"sugar", "brown sugar", "powdered sugar", "glucose", "fructose", "molasses",
		"syrup", "honey", "jaggery", "jam", "jelly", "candy", "candies", "sweetened",
		"soda", "nectar", "condensed milk", "dessert", "cake", "cookie", "cookies",
		"ice cream", "frosting", "waffle", "waffles", "flan",
	) {
		diabetesReasons = append(diabetesReasons, "added_sugar_or_dessert")
		diabetesFlags = append(diabetesFlags, "added_sugar")
	}
	if carbs >= 45 && hasAny(
		"white bread", "white rice", "flour", "maida", "pasta", "noodle", "noodles",
		"tortilla", "burger bun", "crackers", "naan", "kulcha", "paratha",
	) {
		diabetesReasons = append(diabetesReasons, "refined_carbohydrate")
		diabetesFlags = append(diabetesFlags, "refined_carb")
	}
	appendResult("diabetes", diabetesReasons, diabetesFlags)

	hypertensionReasons := []string{}
	hypertensionFlags := []string{}
	if sodium >= 700 {
		hypertensionReasons = append(hypertensionReasons, "sodium_over_limit")
		hypertensionFlags = append(hypertensionFlags, "high_sodium")
	}
	if hasAny(
		"bacon", "sausage", "pancetta", "salami", "pepperoni", "ham", "prosciutto",
		"salted", "pickle", "pickled", "anchovy", "worcestershire", "soy sauce",
		"tamari", "stock cube", "bouillon cube", "processed cheese",
	) {
		hypertensionReasons = append(hypertensionReasons, "salty_or_processed_ingredient")
		hypertensionFlags = append(hypertensionFlags, "salty_processed")
	}
	appendResult("hypertension", hypertensionReasons, hypertensionFlags)

	cardiacReasons := []string{}
	cardiacFlags := []string{}
	if fat >= 24 {
		cardiacReasons = append(cardiacReasons, "fat_over_limit")
		cardiacFlags = append(cardiacFlags, "high_fat")
	}
	if sodium >= 850 {
		cardiacReasons = append(cardiacReasons, "sodium_over_cardiac_limit")
		cardiacFlags = append(cardiacFlags, "high_sodium")
	}
	if calories >= 850 {
		cardiacReasons = append(cardiacReasons, "calories_over_limit")
		cardiacFlags = append(cardiacFlags, "high_calorie")
	}
	if hasAny(
		"fried", "fries", "fritter", "tempura", "onion rings", "deep fried", "pakora",
		"samosa", "puri", "poori", "butter", "cream", "ghee", "lard", "palm oil",
		"coconut oil", "mayonnaise", "shortening", "bacon", "sausage", "pancetta",
		"salami", "pepperoni", "offal", "liver", "kidney", "organ meat",
	) {
		cardiacReasons = append(cardiacReasons, "saturated_fat_fried_or_processed_ingredient")
		cardiacFlags = append(cardiacFlags, "cardiac_risk_ingredient")
	}
	appendResult("cardiac", cardiacReasons, cardiacFlags)

	renalReasons := []string{}
	renalFlags := []string{}
	if protein >= 35 {
		renalReasons = append(renalReasons, "protein_over_renal_limit")
		renalFlags = append(renalFlags, "high_protein")
	}
	if sodium >= 600 {
		renalReasons = append(renalReasons, "sodium_over_renal_limit")
		renalFlags = append(renalFlags, "high_sodium")
	}
	if hasAny(
		"banana", "potato", "sweet potato", "tomato", "spinach", "avocado", "beans",
		"lentils", "molasses", "coconut water", "orange", "dates", "raisin", "raisins",
	) {
		renalReasons = append(renalReasons, "potassium_rich_ingredient")
		renalFlags = append(renalFlags, "potassium_rich")
	}
	if hasAny(
		"milk", "cheese", "yogurt", "nuts", "almond", "walnut", "seeds", "sesame",
		"processed cheese", "cola",
	) {
		renalReasons = append(renalReasons, "phosphorus_rich_ingredient")
		renalFlags = append(renalFlags, "phosphorus_rich")
	}
	if hasAny(
		"anchovy", "sardine", "offal", "liver", "kidney", "bouzelouf", "daouara",
		"organ meat", "tripe", "mutton", "beef",
	) {
		renalReasons = append(renalReasons, "purine_or_offal_ingredient")
		renalFlags = append(renalFlags, "purine_rich")
	}
	appendResult("renal_failure", renalReasons, renalFlags)

	cholesterolReasons := []string{}
	cholesterolFlags := []string{}
	if fat >= 22 {
		cholesterolReasons = append(cholesterolReasons, "fat_over_cholesterol_limit")
		cholesterolFlags = append(cholesterolFlags, "high_fat")
	}
	if hasAny(
		"fried", "fries", "fritter", "tempura", "onion rings", "deep fried", "pakora",
		"samosa", "puri", "poori", "butter", "cream", "ghee", "lard", "palm oil",
		"coconut oil", "mayonnaise", "shortening", "bacon", "sausage", "pancetta",
		"salami", "pepperoni", "offal", "liver", "kidney", "organ meat", "beef",
		"lamb", "mutton",
	) {
		cholesterolReasons = append(cholesterolReasons, "saturated_fat_or_processed_ingredient")
		cholesterolFlags = append(cholesterolFlags, "cholesterol_risk_ingredient")
	}
	appendResult("hypercholesterolemia", cholesterolReasons, cholesterolFlags)

	digestiveReasons := []string{}
	digestiveFlags := []string{}
	if fat >= 24 {
		digestiveReasons = append(digestiveReasons, "fat_over_digestive_limit")
		digestiveFlags = append(digestiveFlags, "high_fat")
	}
	if hasAny(
		"chili", "green chili", "dried chilies", "cayenne", "hot sauce", "jalapeno",
		"red chili powder", "garam masala", "pepper seasoning",
	) {
		digestiveReasons = append(digestiveReasons, "spicy_ingredient")
		digestiveFlags = append(digestiveFlags, "very_spicy")
	}
	if hasAny(
		"beans", "bean", "lentils", "lentil", "chickpea", "chickpeas", "cabbage",
		"broccoli", "cauliflower", "onion", "garlic", "peas", "rajma", "chana",
		"black gram",
	) {
		digestiveReasons = append(digestiveReasons, "gas_forming_ingredient")
		digestiveFlags = append(digestiveFlags, "gas_forming")
	}
	if hasAny("fried", "fries", "fritter", "tempura", "onion rings", "deep fried", "pakora", "samosa") {
		digestiveReasons = append(digestiveReasons, "fried_ingredient")
		digestiveFlags = append(digestiveFlags, "fried")
	}
	appendResult("digestive_sensitivity", digestiveReasons, digestiveFlags)

	return out
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
