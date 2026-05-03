package catalog

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var IngredientLabelsFR = map[string]string{
	"almond":         "Amande",
	"apple":          "Pomme",
	"avocado":        "Avocat",
	"banana":         "Banane",
	"basil":          "Basilic",
	"beef":           "Boeuf",
	"bell pepper":    "Poivron",
	"broccoli":       "Brocoli",
	"brown rice":     "Riz complet",
	"butter":         "Beurre",
	"carrot":         "Carotte",
	"cheese":         "Fromage",
	"chicken":        "Poulet",
	"chicken breast": "Blanc de poulet",
	"chickpea":       "Pois chiche",
	"cinnamon":       "Cannelle",
	"cod":            "Cabillaud",
	"cream":          "Creme",
	"cucumber":       "Concombre",
	"egg":            "Oeuf",
	"eggs":           "Oeufs",
	"flour":          "Farine",
	"garlic":         "Ail",
	"ginger":         "Gingembre",
	"greek yogurt":   "Yaourt grec",
	"green bean":     "Haricot vert",
	"honey":          "Miel",
	"lamb":           "Agneau",
	"lentil":         "Lentille",
	"lettuce":        "Laitue",
	"milk":           "Lait",
	"mushroom":       "Champignon",
	"oat":            "Avoine",
	"olive oil":      "Huile d'olive",
	"onion":          "Oignon",
	"pasta":          "Pates",
	"peanut":         "Arachide",
	"peas":           "Petits pois",
	"pepper":         "Poivre",
	"pork":           "Porc",
	"potato":         "Pomme de terre",
	"quinoa":         "Quinoa",
	"rice":           "Riz",
	"salmon":         "Saumon",
	"salt":           "Sel",
	"shrimp":         "Crevette",
	"spinach":        "Epinard",
	"sugar":          "Sucre",
	"sweet potato":   "Patate douce",
	"tofu":           "Tofu",
	"tomato":         "Tomate",
	"tuna":           "Thon",
	"turkey":         "Dinde",
	"walnut":         "Noix",
	"wheat":          "Ble",
	"whole wheat":    "Ble complet",
	"yogurt":         "Yaourt",
}

var ingredientTermLabelsFR = map[string]string{
	"anchovy":     "anchois",
	"asparagus":   "asperge",
	"beans":       "haricots",
	"bean":        "haricot",
	"bread":       "pain",
	"breast":      "blanc",
	"cabbage":     "chou",
	"cake":        "gateau",
	"cauliflower": "chou-fleur",
	"chili":       "piment",
	"chocolate":   "chocolat",
	"coconut":     "noix de coco",
	"corn":        "mais",
	"crab":        "crabe",
	"dried":       "seche",
	"duck":        "canard",
	"fish":        "poisson",
	"fried":       "frit",
	"fruit":       "fruit",
	"green":       "vert",
	"grilled":     "grille",
	"liver":       "foie",
	"lobster":     "homard",
	"meat":        "viande",
	"minced":      "hache",
	"oil":         "huile",
	"orange":      "orange",
	"powder":      "poudre",
	"red":         "rouge",
	"sauce":       "sauce",
	"seed":        "graine",
	"seeds":       "graines",
	"sesame":      "sesame",
	"soup":        "soupe",
	"spice":       "epice",
	"spices":      "epices",
	"sweet":       "doux",
	"veal":        "veau",
	"vegetable":   "legume",
	"vegetables":  "legumes",
	"white":       "blanc",
	"whole":       "complet",
}

func IngredientDisplayLabel(value string) string {
	normalized := NormalizeIngredientValue(value)
	if normalized == "" {
		return ""
	}
	if label, ok := IngredientLabelsFR[normalized]; ok {
		return label
	}
	return titleCaseFR(translateKnownIngredientTerms(normalized))
}

func IngredientSearchTerms(query string) []string {
	normalized := NormalizeIngredientValue(query)
	foldedQuery := foldFrenchSearch(normalized)
	if len(foldedQuery) < 2 {
		return nil
	}

	terms := make([]string, 0, 6)
	seen := map[string]struct{}{}
	add := func(value string) {
		value = NormalizeIngredientValue(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		terms = append(terms, value)
	}

	// Prefer canonical catalog keys whose French UI label matches the user query.
	labelKeys := make([]string, 0, len(IngredientLabelsFR))
	for key := range IngredientLabelsFR {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)
	for _, key := range labelKeys {
		label := foldFrenchSearch(NormalizeIngredientValue(IngredientLabelsFR[key]))
		if label == "" {
			continue
		}
		if strings.Contains(label, foldedQuery) || strings.Contains(foldedQuery, label) || containsAllSearchTokens(label, foldedQuery) {
			add(key)
		}
	}

	termKeys := make([]string, 0, len(ingredientTermLabelsFR))
	for key := range ingredientTermLabelsFR {
		termKeys = append(termKeys, key)
	}
	sort.Strings(termKeys)
	for _, key := range termKeys {
		label := foldFrenchSearch(NormalizeIngredientValue(ingredientTermLabelsFR[key]))
		if label == "" {
			continue
		}
		if strings.Contains(label, foldedQuery) || strings.Contains(foldedQuery, label) || containsAllSearchTokens(label, foldedQuery) {
			add(key)
		}
	}

	add(normalized)
	add(foldedQuery)
	return terms
}

func NormalizeIngredientValue(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	if lowered == "" {
		return ""
	}
	mapped := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return r
		default:
			return ' '
		}
	}, lowered)
	return strings.Join(strings.Fields(mapped), " ")
}

func foldFrenchSearch(value string) string {
	replacer := strings.NewReplacer(
		"œ", "oe", "Œ", "oe", "æ", "ae", "Æ", "ae",
		"à", "a", "á", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
		"ç", "c",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ñ", "n",
		"ò", "o", "ó", "o", "ô", "o", "ö", "o", "õ", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ý", "y", "ÿ", "y",
	)
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), " ")
}

func containsAllSearchTokens(label, query string) bool {
	queryTokens := strings.Fields(query)
	if len(queryTokens) == 0 {
		return false
	}
	for _, token := range queryTokens {
		if token == "d" || token == "de" || token == "du" || token == "des" || token == "a" {
			continue
		}
		if !strings.Contains(label, token) {
			return false
		}
	}
	return true
}

func translateKnownIngredientTerms(value string) string {
	terms := make([]string, 0, len(ingredientTermLabelsFR))
	for term := range ingredientTermLabelsFR {
		terms = append(terms, term)
	}
	sort.Slice(terms, func(i, j int) bool {
		return len(terms[i]) > len(terms[j])
	})

	translated := value
	for _, term := range terms {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
		translated = re.ReplaceAllString(translated, ingredientTermLabelsFR[term])
	}
	return translated
}

func titleCaseFR(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
