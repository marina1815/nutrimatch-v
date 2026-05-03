package catalog

import (
	"sort"
	"strings"
	"unicode"
)

// InferSafetyTags derives deterministic health and recipe tags from local catalog
// facts. These tags are advisory metadata for rules/scoring; they never replace
// ingredient-level hard filters.
func InferSafetyTags(title, description string, ingredients []string, calories, protein, carbs, fat, sugar, sodium float64) []string {
	text := normalizeTagText(title + " " + description + " " + strings.Join(ingredients, " "))
	ingredientCount := len(normalizeIngredientsForTags(ingredients))
	tags := map[string]struct{}{}

	add := func(values ...string) {
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				tags[value] = struct{}{}
			}
		}
	}
	hasAny := func(patterns ...string) bool {
		for _, pattern := range patterns {
			if containsTagTerm(text, pattern) {
				return true
			}
		}
		return false
	}

	if hasAny("salad", "lettuce", "cucumber", "mixed greens", "romaine", "spinach") {
		add("salad", "vegetable-rich", "healthy")
	}
	if hasAny("soup", "broth", "chorba", "harira") {
		add("soup")
	}
	if hasAny("oat", "egg", "pancake", "cereal", "granola", "breakfast") {
		add("breakfast")
	}
	if hasAny("dessert", "cake", "cookie", "cookies", "frosting", "candy", "candies", "ice cream", "flan", "waffle", "waffles", "chocolate", "jam", "syrup", "jaggery", "honey", "condensed milk", "sweet chutney") {
		add("dessert", "sugary", "diabetes-risk")
	}
	if hasAny("sugar", "brown sugar", "palm sugar", "molasses", "glucose", "fructose", "sweetened", "soda", "nectar", "syrup", "honey", "jaggery", "powdered sugar") {
		add("sugary", "sweetened", "diabetes-risk")
	}
	if sugar >= 18 || carbs >= 65 {
		add("high-sugar", "diabetes-risk")
	}
	if sugar > 0 && sugar <= 12 && carbs <= 55 {
		add("low-sugar")
	}
	if hasAny("white bread", "white rice", "flour", "maida", "pasta", "tortilla", "burger bun", "crackers", "noodle", "noodles", "naan", "kulcha", "paratha") && carbs >= 45 {
		add("refined-carb", "diabetes-risk")
	}

	if sodium >= 800 {
		add("high-sodium", "salty", "hypertension-risk")
	}
	if sodium > 0 && sodium <= 650 {
		add("low-sodium")
	}
	if hasAny("bacon", "sausage", "pancetta", "salted", "pickle", "pickled", "anchovy", "worcestershire", "soy sauce", "tamari", "stock cube", "bouillon cube", "processed cheese", "salami", "pepperoni") {
		add("salty", "processed-meat", "hypertension-risk")
	}

	if fat >= 24 || calories >= 850 {
		add("high-fat", "cardiac-risk")
	}
	if fat > 0 && fat <= 16 {
		add("low-fat")
	}
	if hasAny("fried", "fries", "fritter", "tempura", "onion rings", "deep fried", "pakora", "samosa", "puri", "poori") {
		add("fried", "high-fat", "cardiac-risk", "cholesterol-risk", "digestive-risk")
	}
	if hasAny("butter", "cream", "cheese", "ghee", "lard", "palm oil", "coconut oil", "pancetta", "bacon", "sausage", "mayonnaise", "shortening", "malai") {
		add("saturated-fat", "cholesterol-risk", "cardiac-risk")
	}
	if hasAny("beef", "red meat", "lamb", "mutton", "offal", "liver", "kidney", "pork", "daouara", "bouzelouf", "tripe", "organ meat", "goat meat") {
		add("red-meat", "cholesterol-risk", "cardiac-risk")
	}

	if protein >= 30 {
		add("high-protein")
	} else if protein >= 18 {
		add("moderate-protein")
	} else if protein > 0 {
		add("low-protein")
	}
	if hasAny("chicken", "turkey", "egg", "eggs", "tofu", "fish", "salmon", "tuna", "beans", "lentils", "chickpea", "beef") {
		add("protein-source")
		if protein >= 18 {
			add("high-protein")
		}
	}
	if hasAny("chicken", "turkey", "poultry") {
		add("poultry")
	}
	if hasAny("fish", "salmon", "tuna", "anchovy", "shrimp", "crab", "lobster", "seafood") {
		add("seafood")
	}
	if hasAny("beans", "bean", "lentils", "lentil", "chickpea", "peas", "soy", "tofu") {
		add("legume")
	}

	if hasAny("banana", "potato", "sweet potato", "tomato", "spinach", "avocado", "beans", "lentils", "molasses", "coconut water", "orange", "dates", "raisin") {
		add("potassium-rich", "renal-risk")
	}
	if hasAny("milk", "cheese", "yogurt", "nuts", "almond", "walnut", "seeds", "sesame", "organ meat", "liver", "cola", "processed cheese") {
		add("phosphorus-rich", "renal-risk")
	}
	if hasAny("anchovy", "sardine", "offal", "liver", "kidney", "bouzelouf", "daouara", "mutton", "beef", "organ meat") {
		add("purine-rich", "renal-risk")
	}
	if hasAny("spinach", "kale", "parsley", "cabbage", "broccoli", "brussels sprout", "collard greens") {
		add("vitamin-k-rich")
	}

	gasScore := countTagMatches(text, "beans", "bean", "lentils", "lentil", "chickpea", "cabbage", "broccoli", "cauliflower", "onion", "garlic", "peas", "rajma", "chana", "black gram")
	if gasScore >= 2 {
		add("gas-forming", "digestive-risk")
	}
	if hasAny("chili", "green chili", "dried chilies", "cayenne", "hot sauce", "pepper seasoning", "garam masala", "red chili powder", "jalapeno") {
		add("very-spicy", "digestive-risk")
	}
	if hasAny("milk", "cream", "cheese", "yogurt") {
		add("dairy")
	}
	if !hasAny("chili", "cayenne", "fried") && fat <= 18 && gasScore == 0 {
		add("gentle-digestive")
	}

	if hasAny("whole grain", "whole wheat", "brown rice", "oat", "quinoa", "barley") {
		add("whole-grain", "healthy")
	}
	if calories > 0 && calories <= 750 && protein >= 18 && sugar <= 18 && sodium <= 800 {
		add("balanced")
	}
	if calories > 0 && calories <= 550 && fat <= 18 {
		add("light")
	}
	if ingredientCount >= 4 && !hasAny("sugar", "fried") {
		add("varied")
	}

	out := make([]string, 0, len(tags))
	for tag := range tags {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func normalizeIngredientsForTags(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := normalizeTagText(value)
		if cleaned != "" && cleaned != "none" {
			out = append(out, cleaned)
		}
	}
	return out
}

func normalizeTagText(input string) string {
	lowered := strings.ToLower(strings.TrimSpace(input))
	if lowered == "" {
		return ""
	}
	mapped := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, lowered)
	return strings.Join(strings.Fields(mapped), " ")
}

func containsTagTerm(text, pattern string) bool {
	pattern = normalizeTagText(pattern)
	if text == "" || pattern == "" {
		return false
	}
	return strings.Contains(" "+text+" ", " "+pattern+" ")
}

func countTagMatches(text string, patterns ...string) int {
	count := 0
	for _, pattern := range patterns {
		if containsTagTerm(text, pattern) {
			count++
		}
	}
	return count
}
