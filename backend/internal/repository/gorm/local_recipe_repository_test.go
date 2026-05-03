package gormrepo

import "testing"

func TestIsLowQualityMealCandidateRejectsPantryItems(t *testing.T) {
	cases := []struct {
		title       string
		ingredients []string
	}{
		{"Apple", []string{"apple"}},
		{"Hazelnut Spread", []string{"sugar", "hazelnuts", "cocoa"}},
		{"Natural Flavorings", []string{"natural flavorings"}},
		{"Sesame Seeds", []string{"sesame seeds"}},
		{"Green Smoothie", []string{"spinach", "honey", "avocado", "almond milk"}},
		{"Mixed Nuts", []string{"nuts", "almonds", "hazelnuts", "pistachios"}},
		{"Gelatinized Starch", []string{"wheat", "amylose", "amylopectin"}},
		{"Calcium-Enriched Juices", []string{"juice", "calcium"}},
		{"Malt Vinegar", []string{"barley", "water"}},
		{"Marinades", []string{"oil", "lemon juice", "spices"}},
		{"Dried Teas", []string{"tea leaves"}},
		{"Cheeses (With Lysozymes)", []string{"milk", "lysozymes"}},
		{"Dried Fruits", []string{"apple", "apricots", "raisins"}},
		{"Mono- and Diglycerides", []string{"fatty acids", "glycerol"}},
		{"Flavors", []string{"flavors"}},
		{"Pesto", []string{"basil", "oil", "nuts"}},
		{"Pulp", []string{"fruit pulp"}},
		{"Muesli Crackers", []string{"muesli", "flour", "sugar"}},
		{"Ground Mustard", []string{"mustard"}},
		{"Salads", []string{"lettuce", "tomato"}},
		{"Parmesan Melted Breadcrumb", []string{"grated parmesan", "salt", "water", "wheat"}},
		{"Matzo", []string{"flour", "salt", "water"}},
		{"Crudites", []string{"raw vegetables carrot", "celery", "pepper"}},
		{"Onion Rings", []string{"flour", "onion", "vegetable oil"}},
		{"Relish Sauce", []string{"pickled vegetables", "spices", "seasonings"}},
		{"Textured Soy Flour", []string{"emulsifier", "soy flour", "water"}},
		{"Frozen Peeled Tomatoes", []string{"peeled tomatoes", "preservative", "salt"}},
		{"Tomato Purees", []string{"tomatoes", "salt", "pepper"}},
		{"Emulsifiers", []string{"soy lecithin", "mono- and diglycerides"}},
		{"Croutons", []string{"bread", "oil", "seasoning"}},
		{"Lecithin", []string{"soy lecithin"}},
		{"Lobster Bisque", []string{"lobster", "cream", "broth"}},
		{"Seitan", []string{"wheat gluten", "water", "seasoning"}},
		{"Bread", []string{"flour", "water", "salt"}},
		{"Mashed Potatoes", []string{"potatoes", "milk", "butter"}},
	}

	for _, tc := range cases {
		if !isLowQualityMealCandidate(tc.title, tc.ingredients) {
			t.Fatalf("expected %q to be excluded from main-meal recommendations", tc.title)
		}
	}
}

func TestIsLowQualityMealCandidateKeepsPlausibleMainMeals(t *testing.T) {
	if isLowQualityMealCandidate("Chana Masala", []string{"chickpeas", "tomato", "onion", "spices"}) {
		t.Fatalf("expected complete savory dish to remain eligible")
	}
}
