package gormrepo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/marina1815/nutrimatch/internal/catalog"
	"github.com/marina1815/nutrimatch/internal/localdata"
	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
)

const defaultLocalRecipeLimit = 10000

type LocalRecipeRepository struct {
	db *gorm.DB
}

func NewLocalRecipeRepository(db *gorm.DB) *LocalRecipeRepository {
	return &LocalRecipeRepository{db: db}
}

func (r *LocalRecipeRepository) Seed(ctx context.Context, seed *localdata.CatalogSeed) error {
	if r == nil || r.db == nil || seed == nil {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM catalog.local_cross_allergies`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_ingredient_allergies`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_recipe_ingredients`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_recipe_medical_compatibility`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_recipes`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_allergies`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM catalog.local_ingredients`).Error; err != nil {
			return err
		}

		for _, ingredient := range seed.Ingredients {
			ingredientKey := normalizeLocalKey(ingredient.Key)
			displayName := strings.TrimSpace(ingredient.DisplayName)
			if ingredientKey == "" || displayName == "" {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_ingredients (ingredient_key, display_name, source, updated_at)
				VALUES (?, ?, ?, now())
				ON CONFLICT (ingredient_key) DO UPDATE SET
					display_name = EXCLUDED.display_name,
					source = EXCLUDED.source,
					updated_at = now()
			`, ingredientKey, displayName, localSource(ingredient.Source)).Error; err != nil {
				return err
			}
		}

		for _, allergy := range seed.Allergies {
			allergyKey := normalizeLocalKey(allergy.Key)
			displayName := strings.TrimSpace(allergy.DisplayName)
			if allergyKey == "" || displayName == "" {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_allergies (allergy_key, display_name, source, updated_at)
				VALUES (?, ?, ?, now())
				ON CONFLICT (allergy_key) DO UPDATE SET
					display_name = EXCLUDED.display_name,
					source = EXCLUDED.source,
					updated_at = now()
			`, allergyKey, displayName, localSource(allergy.Source)).Error; err != nil {
				return err
			}
		}

		for _, recipe := range seed.Recipes {
			if strings.TrimSpace(recipe.ID) == "" || strings.TrimSpace(recipe.Title) == "" {
				continue
			}
			protein := calibratedLocalProtein(recipe.Title, recipe.Ingredients, recipe.Nutrition.Protein)
			tags := catalog.InferSafetyTags(recipe.Title, "", recipe.Ingredients, recipe.Nutrition.Calories, protein, recipe.Nutrition.Carbs, recipe.Nutrition.Fat, recipe.Nutrition.Sugar, recipe.Nutrition.SodiumMg)
			medicalCompatibility := catalog.AssessMedicalCompatibility(recipe.Title, recipe.Ingredients, recipe.Nutrition.Calories, protein, recipe.Nutrition.Carbs, recipe.Nutrition.Fat, recipe.Nutrition.Sugar, recipe.Nutrition.SodiumMg)
			payload, _ := json.Marshal(map[string]any{
				"source":               recipe.Source,
				"ingredients":          recipe.Ingredients,
				"tags":                 tags,
				"medicalCompatibility": medicalCompatibility,
				"seedVersion":          seed.Version,
			})
			if err := tx.Exec(`
				INSERT INTO catalog.local_recipes (
					id, title, source, calories, protein, carbs, fat, sugar, sodium_mg, payload, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS jsonb), now())
				ON CONFLICT (id) DO UPDATE SET
					title = EXCLUDED.title,
					source = EXCLUDED.source,
					calories = EXCLUDED.calories,
					protein = EXCLUDED.protein,
					carbs = EXCLUDED.carbs,
					fat = EXCLUDED.fat,
					sugar = EXCLUDED.sugar,
					sodium_mg = EXCLUDED.sodium_mg,
					payload = EXCLUDED.payload,
					updated_at = now()
			`, recipe.ID, recipe.Title, localSource(recipe.Source), recipe.Nutrition.Calories, protein, recipe.Nutrition.Carbs, recipe.Nutrition.Fat, recipe.Nutrition.Sugar, recipe.Nutrition.SodiumMg, string(payload)).Error; err != nil {
				return err
			}
			for _, compatibility := range medicalCompatibility {
				reasons, _ := json.Marshal(compatibility.Reasons)
				riskFlags, _ := json.Marshal(compatibility.RiskFlags)
				if err := tx.Exec(`
					INSERT INTO catalog.local_recipe_medical_compatibility (
						recipe_id, condition_key, compatible, reasons, risk_flags, source
					) VALUES (?, ?, ?, CAST(? AS jsonb), CAST(? AS jsonb), ?)
					ON CONFLICT (recipe_id, condition_key) DO UPDATE SET
						compatible = EXCLUDED.compatible,
						reasons = EXCLUDED.reasons,
						risk_flags = EXCLUDED.risk_flags,
						source = EXCLUDED.source
				`, recipe.ID, compatibility.ConditionKey, compatibility.Compatible, string(reasons), string(riskFlags), catalog.MedicalCompatibilitySource).Error; err != nil {
					return err
				}
			}
			for _, ingredient := range recipe.Ingredients {
				key := normalizeLocalKey(ingredient)
				if key == "" || key == "none" {
					continue
				}
				if err := tx.Exec(`
					INSERT INTO catalog.local_ingredients (ingredient_key, display_name, source, updated_at)
					VALUES (?, ?, ?, now())
					ON CONFLICT (ingredient_key) DO UPDATE SET updated_at = now()
				`, key, ingredient, localSource(recipe.Source)).Error; err != nil {
					return err
				}
				if err := tx.Exec(`
					INSERT INTO catalog.local_recipe_ingredients (recipe_id, ingredient_key)
					VALUES (?, ?)
					ON CONFLICT (recipe_id, ingredient_key) DO NOTHING
				`, recipe.ID, key).Error; err != nil {
					return err
				}
			}
		}

		for _, allergy := range seed.IngredientAllergies {
			ingredient := normalizeLocalKey(allergy.IngredientKey)
			allergyKey := normalizeLocalKey(allergy.AllergyKey)
			if ingredient == "" || allergyKey == "" {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_ingredients (ingredient_key, display_name, source, updated_at)
				VALUES (?, ?, ?, now())
				ON CONFLICT (ingredient_key) DO UPDATE SET updated_at = now()
			`, ingredient, ingredient, localSource(allergy.Source)).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_allergies (allergy_key, display_name, source, updated_at)
				VALUES (?, ?, ?, now())
				ON CONFLICT (allergy_key) DO UPDATE SET updated_at = now()
			`, allergyKey, allergy.AllergyKey, localSource(allergy.Source)).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_ingredient_allergies (ingredient_key, allergy_key, source)
				VALUES (?, ?, ?)
				ON CONFLICT (ingredient_key, allergy_key) DO UPDATE SET source = EXCLUDED.source
			`, ingredient, allergyKey, localSource(allergy.Source)).Error; err != nil {
				return err
			}
		}

		for _, cross := range seed.CrossAllergies {
			ingredient := normalizeLocalKey(cross.IngredientKey)
			crossIngredient := normalizeLocalKey(cross.CrossIngredientKey)
			allergyKey := normalizeLocalKey(cross.AllergyKey)
			if ingredient == "" || crossIngredient == "" || allergyKey == "" {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_ingredients (ingredient_key, display_name, source, updated_at)
				VALUES (?, ?, 'croise.xlsx', now())
				ON CONFLICT (ingredient_key) DO UPDATE SET updated_at = now()
			`, ingredient, ingredient).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_ingredients (ingredient_key, display_name, source, updated_at)
				VALUES (?, ?, 'croise.xlsx', now())
				ON CONFLICT (ingredient_key) DO UPDATE SET updated_at = now()
			`, crossIngredient, crossIngredient).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_allergies (allergy_key, display_name, source, updated_at)
				VALUES (?, ?, 'croise.xlsx', now())
				ON CONFLICT (allergy_key) DO UPDATE SET updated_at = now()
			`, allergyKey, cross.AllergyKey).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO catalog.local_cross_allergies (ingredient_key, cross_ingredient_key, allergy_key)
				VALUES (?, ?, ?)
				ON CONFLICT (ingredient_key, cross_ingredient_key, allergy_key) DO NOTHING
			`, ingredient, crossIngredient, allergyKey).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *LocalRecipeRepository) Search(ctx context.Context, query repository.LocalRecipeQuery) ([]repository.LocalRecipeCandidate, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	var rows []localRecipeSearchRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			lr.id,
			lr.title,
			lr.calories,
			lr.protein,
			lr.carbs,
			lr.fat,
			lr.sugar,
			lr.sodium_mg,
			lr.payload,
			COALESCE(jsonb_agg(lri.ingredient_key ORDER BY lri.ingredient_key) FILTER (WHERE lri.ingredient_key IS NOT NULL), '[]'::jsonb) AS ingredients_json
		FROM catalog.local_recipes lr
		LEFT JOIN catalog.local_recipe_ingredients lri ON lri.recipe_id = lr.id
		GROUP BY lr.id
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	allergyMap, err := r.loadIngredientAllergyMap(ctx)
	if err != nil {
		return nil, err
	}
	crossMap, err := r.loadCrossAllergyMap(ctx)
	if err != nil {
		return nil, err
	}
	medicalMap, err := r.loadMedicalCompatibilityMap(ctx, query.ConditionKeys)
	if err != nil {
		return nil, err
	}

	blockedIngredients := expandBlockedIngredients(query.ExcludedIngredients, query.AllergyKeys)
	allergyKeys := normalizeSet(query.AllergyKeys)
	conditionKeys := normalizeConditionSet(query.ConditionKeys)
	terms := mergeNormalized(query.QueryTerms, query.Likes)

	candidates := make([]repository.LocalRecipeCandidate, 0, len(rows))
	for _, row := range rows {
		ingredients := decodeIngredientKeys(row.IngredientsJSON)
		if localRecipeBlocked(row.Title, ingredients, blockedIngredients, allergyKeys, allergyMap, crossMap) {
			continue
		}
		compatibility, riskFlags, blocked := recipeMedicalCompatibility(row.ID, conditionKeys, medicalMap)
		if blocked {
			continue
		}

		score := localRecipeScore(row.Title, ingredients, terms, normalizeSet(query.Likes))
		protein := calibratedLocalProtein(row.Title, ingredients, row.Protein)
		tags := decodePayloadTags(row.Payload)
		if len(tags) == 0 {
			tags = catalog.InferSafetyTags(row.Title, "", ingredients, row.Calories, protein, row.Carbs, row.Fat, row.Sugar, row.SodiumMg)
		}
		if isLowQualityMealCandidate(row.Title, ingredients) {
			continue
		}
		candidates = append(candidates, repository.LocalRecipeCandidate{
			ID:                   row.ID,
			Title:                row.Title,
			Ingredients:          ingredients,
			Tags:                 tags,
			Calories:             row.Calories,
			Protein:              protein,
			Carbs:                row.Carbs,
			Fat:                  row.Fat,
			Sugar:                row.Sugar,
			SodiumMg:             row.SodiumMg,
			Score:                score,
			MedicalCompatibility: compatibility,
			MedicalRiskFlags:     riskFlags,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Title < candidates[j].Title
		}
		return candidates[i].Score > candidates[j].Score
	})

	limit := query.Limit
	if limit <= 0 {
		limit = defaultLocalRecipeLimit
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func (r *LocalRecipeRepository) SuggestIngredients(ctx context.Context, query string, limit int) ([]repository.CatalogOption, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	cleaned := normalizeLocalKey(query)
	if len(cleaned) < 2 {
		return []repository.CatalogOption{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}

	type ingredientSuggestionRow struct {
		Value       string `gorm:"column:value"`
		DisplayName string `gorm:"column:display_name"`
	}

	searchTerms := catalog.IngredientSearchTerms(query)
	if len(searchTerms) == 0 {
		searchTerms = []string{cleaned}
	}

	out := make([]repository.CatalogOption, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, term := range searchTerms {
		term = normalizeLocalKey(term)
		if term == "" {
			continue
		}

		like := "%" + term + "%"
		var rows []ingredientSuggestionRow
		if err := r.db.WithContext(ctx).Raw(`
			SELECT ingredient_key AS value, display_name
			FROM catalog.local_ingredients
			WHERE ingredient_key LIKE ? OR lower(display_name) LIKE ?
			ORDER BY
				CASE
					WHEN ingredient_key = ? THEN 0
					WHEN ingredient_key LIKE ? THEN 1
					ELSE 2
				END,
				length(display_name),
				display_name
			LIMIT ?
		`, like, like, term, term+"%", limit).Scan(&rows).Error; err != nil {
			return nil, err
		}

		for _, row := range rows {
			value := normalizeLocalKey(row.Value)
			if value == "" {
				value = normalizeLocalKey(row.DisplayName)
			}
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, repository.CatalogOption{
				Value:  value,
				Label:  catalog.IngredientDisplayLabel(value),
				Source: "local_catalog",
			})
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (r *LocalRecipeRepository) ListAllergies(ctx context.Context) ([]repository.CatalogOption, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	var rows []struct {
		Value  string `gorm:"column:value"`
		Label  string `gorm:"column:label"`
		Source string `gorm:"column:source"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT allergy_key AS value, display_name AS label, source
		FROM catalog.local_allergies
		WHERE allergy_key <> '' AND display_name <> ''
		ORDER BY
			CASE
				WHEN source = 'liste des allergies.xlsx' THEN 0
				WHEN source = 'ingrediant_EN.xlsx' THEN 1
				ELSE 2
			END,
			display_name
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]repository.CatalogOption, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		value := normalizeLocalKey(row.Value)
		label := strings.TrimSpace(row.Label)
		if value == "" || label == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, repository.CatalogOption{
			Value:  value,
			Label:  label,
			Source: localSource(row.Source),
		})
	}
	return out, nil
}

type localRecipeSearchRow struct {
	ID              string
	Title           string
	Calories        float64
	Protein         float64
	Carbs           float64
	Fat             float64
	Sugar           float64
	SodiumMg        float64 `gorm:"column:sodium_mg"`
	Payload         []byte
	IngredientsJSON []byte `gorm:"column:ingredients_json"`
}

type localIngredientAllergyRow struct {
	IngredientKey string
	AllergyKey    string
}

type localCrossAllergyRow struct {
	IngredientKey      string
	CrossIngredientKey string
	AllergyKey         string
}

type localMedicalCompatibilityRow struct {
	RecipeID      string `gorm:"column:recipe_id"`
	ConditionKey  string `gorm:"column:condition_key"`
	Compatible    bool
	ReasonsJSON   []byte `gorm:"column:reasons"`
	RiskFlagsJSON []byte `gorm:"column:risk_flags"`
}

type localMedicalCompatibility struct {
	Compatible bool
	Reasons    []string
	RiskFlags  []string
}

func (r *LocalRecipeRepository) loadIngredientAllergyMap(ctx context.Context) (map[string]map[string]struct{}, error) {
	var rows []localIngredientAllergyRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT ingredient_key, allergy_key
		FROM catalog.local_ingredient_allergies
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]map[string]struct{}, len(rows))
	for _, row := range rows {
		ingredient := normalizeLocalKey(row.IngredientKey)
		allergy := normalizeLocalKey(row.AllergyKey)
		if ingredient == "" || allergy == "" {
			continue
		}
		if out[ingredient] == nil {
			out[ingredient] = map[string]struct{}{}
		}
		out[ingredient][allergy] = struct{}{}
	}
	return out, nil
}

func (r *LocalRecipeRepository) loadCrossAllergyMap(ctx context.Context) (map[string]map[string]struct{}, error) {
	var rows []localCrossAllergyRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT ingredient_key, cross_ingredient_key, allergy_key
		FROM catalog.local_cross_allergies
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]map[string]struct{}, len(rows))
	for _, row := range rows {
		ingredient := normalizeLocalKey(row.IngredientKey)
		crossIngredient := normalizeLocalKey(row.CrossIngredientKey)
		if ingredient == "" || crossIngredient == "" {
			continue
		}
		if out[ingredient] == nil {
			out[ingredient] = map[string]struct{}{}
		}
		out[ingredient][crossIngredient] = struct{}{}
	}
	return out, nil
}

func (r *LocalRecipeRepository) loadMedicalCompatibilityMap(ctx context.Context, conditions []string) (map[string]map[string]localMedicalCompatibility, error) {
	conditionSet := normalizeConditionSet(conditions)
	if len(conditionSet) == 0 {
		return map[string]map[string]localMedicalCompatibility{}, nil
	}

	conditionList := mapKeysSorted(conditionSet)
	var rows []localMedicalCompatibilityRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT recipe_id, condition_key, compatible, reasons, risk_flags
		FROM catalog.local_recipe_medical_compatibility
		WHERE condition_key IN ?
	`, conditionList).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[string]map[string]localMedicalCompatibility, len(rows))
	for _, row := range rows {
		recipeID := strings.TrimSpace(row.RecipeID)
		condition := normalizeConditionKey(row.ConditionKey)
		if recipeID == "" || condition == "" {
			continue
		}
		if out[recipeID] == nil {
			out[recipeID] = map[string]localMedicalCompatibility{}
		}
		out[recipeID][condition] = localMedicalCompatibility{
			Compatible: row.Compatible,
			Reasons:    decodeStringJSONList(row.ReasonsJSON),
			RiskFlags:  decodeStringJSONList(row.RiskFlagsJSON),
		}
	}
	return out, nil
}

func recipeMedicalCompatibility(recipeID string, conditions map[string]struct{}, compatibilityByRecipe map[string]map[string]localMedicalCompatibility) (map[string]bool, []string, bool) {
	out := map[string]bool{}
	flags := map[string]struct{}{}
	if len(conditions) == 0 {
		return out, nil, false
	}

	byCondition := compatibilityByRecipe[recipeID]
	for condition := range conditions {
		compatibility, exists := byCondition[condition]
		if !exists {
			return out, nil, true
		}
		out[condition] = compatibility.Compatible
		for _, flag := range compatibility.RiskFlags {
			flag = strings.TrimSpace(flag)
			if flag != "" {
				flags[flag] = struct{}{}
			}
		}
		if !compatibility.Compatible {
			return out, mapKeysSorted(flags), true
		}
	}
	return out, mapKeysSorted(flags), false
}

func localRecipeBlocked(title string, ingredients []string, blockedIngredients, allergyKeys map[string]struct{}, allergyMap, crossMap map[string]map[string]struct{}) bool {
	terms := make([]string, 0, len(ingredients)+1)
	terms = append(terms, title)
	terms = append(terms, ingredients...)
	for _, ingredient := range terms {
		key := normalizeLocalKey(ingredient)
		if key == "" {
			continue
		}
		if _, blocked := blockedIngredients[key]; blocked {
			return true
		}
		for blocked := range blockedIngredients {
			if blocked != "" && (strings.Contains(key, blocked) || strings.Contains(blocked, key)) {
				return true
			}
		}
		for allergy := range allergyKeys {
			if ingredientMatchesAllergy(key, allergy, allergyMap) {
				return true
			}
		}
		for related := range crossMap[key] {
			if _, blocked := blockedIngredients[related]; blocked {
				return true
			}
		}
	}
	return false
}

func isLowQualityMealCandidate(title string, ingredients []string) bool {
	normalizedIngredients := make([]string, 0, len(ingredients))
	for _, ingredient := range ingredients {
		key := normalizeLocalKey(ingredient)
		if key != "" && key != "none" {
			normalizedIngredients = append(normalizedIngredients, key)
		}
	}
	titleKey := normalizeLocalKey(title)
	if titleKey == "" {
		return true
	}
	if isExactLowQualityTitle(titleKey) {
		return true
	}
	if hasAnyLocalTerm(titleKey, []string{
		"drink", "juice", "juices", "smoothie", "kefir", "beverage",
		"mixed nuts", "nuts", "sprout", "sprouts", "starch", "starches", "gelatinized starch",
		"synthetic meat", "synthetic meats", "natural flavoring", "natural flavorings",
		"oil", "oils", "seed", "seeds", "salsa", "sauce", "paste", "spread", "dressing",
		"vinegar", "marinade", "marinades", "dried tea", "tea", "teas", "cheese", "cheeses",
		"curd", "lysozyme", "lysozymes", "dried fruit", "dried fruits", "yogurt", "flavor",
		"flavors", "flavour", "flavours", "diglyceride", "diglycerides", "biscuit", "biscuits",
		"cracker", "crackers", "pulp", "raisins", "barley", "apple", "sugar", "salt",
		"breadcrumb", "breadcrumbs", "matzo", "crudite", "crudites", "onion rings",
		"relish", "tahini", "soy flour", "peeled tomato", "peeled tomatoes", "tomato puree", "tomato purees",
		"juice", "cider", "butter", "ghee", "syrup", "extract", "extracts", "drink mix",
		"flavored coffee", "coffee substitute", "cornstarch", "cream of tartar", "cream",
		"caramel", "soybeans", "sesame bars", "chocolate bars", "emulsifier", "emulsifiers",
		"broth", "broths", "dried apricot", "dried apricots", "substitute", "substitutes",
		"sprout", "sprouts", "pectin", "dehydrated potato", "dehydrated potatoes",
		"crouton", "croutons", "lecithin", "bisque", "seitan", "mashed potatoes", "french fries",
		"breaded foods", "imitation meat products",
	}) {
		return true
	}
	if len(normalizedIngredients) <= 1 {
		return true
	}
	if len(normalizedIngredients) <= 3 && hasAnyLocalTerm(titleKey, []string{
		"oil", "oils", "flavoring", "flavorings", "extract", "extracts", "seed", "seeds",
		"spice", "spices", "seasoning", "sauce", "salsa", "paste", "spread", "dressing",
		"raisins", "barley", "apple", "sugar", "salt",
	}) {
		return true
	}
	if len(normalizedIngredients) <= 3 {
		for _, ingredient := range normalizedIngredients {
			if titleKey == ingredient {
				return true
			}
		}
	}
	return false
}

func isExactLowQualityTitle(titleKey string) bool {
	switch titleKey {
	case "pesto", "taramosalata", "flavors", "flavours", "mono and diglycerides", "dried fruits", "dried teas", "curries", "pulp", "ground mustard", "salads", "matzo", "crudites", "onion rings", "bread", "mashed potatoes", "french fries", "frozen french fries", "breaded foods", "herbs", "chicory", "tempeh", "imitation meat products", "sandwiches":
		return true
	default:
		return false
	}
}

func ingredientMatchesAllergy(ingredient, allergy string, allergyMap map[string]map[string]struct{}) bool {
	if ingredient == "" || allergy == "" {
		return false
	}
	if ingredient == allergy || strings.Contains(ingredient, allergy) || strings.Contains(allergy, ingredient) {
		return true
	}
	if mapped := allergyMap[ingredient]; mapped != nil {
		if _, ok := mapped[allergy]; ok {
			return true
		}
		for mappedAllergy := range mapped {
			if strings.Contains(mappedAllergy, allergy) || strings.Contains(allergy, mappedAllergy) {
				return true
			}
		}
	}
	return false
}

func setHasAny(values map[string]struct{}, expected ...string) bool {
	for _, value := range expected {
		if _, ok := values[normalizeLocalKey(value)]; ok {
			return true
		}
	}
	return false
}

func localRecipeScore(title string, ingredients []string, terms, likes map[string]struct{}) float64 {
	score := 10.0
	text := normalizeLocalKey(title + " " + strings.Join(ingredients, " "))
	for term := range terms {
		if term != "" && strings.Contains(text, term) {
			score += 12
		}
	}
	for like := range likes {
		if like != "" && strings.Contains(text, like) {
			score += 18
		}
	}
	if strings.Contains(text, "chicken") || strings.Contains(text, "fish") || strings.Contains(text, "beef") || strings.Contains(text, "tofu") {
		score += 5
	}
	return score
}

func calibratedLocalProtein(title string, ingredients []string, protein float64) float64 {
	text := normalizeLocalKey(title + " " + strings.Join(ingredients, " "))
	if hasAnyLocalTerm(text, []string{
		"chicken", "turkey", "beef", "veal", "lamb", "fish", "tuna", "salmon", "cod",
		"shrimp", "prawn", "crab", "lobster", "egg", "eggs", "pork", "ham", "sardine",
		"meat", "steak", "kefta", "kofta", "kebab",
	}) && protein < 52 {
		return 52
	}
	if hasAnyLocalTerm(text, []string{
		"tofu", "lentil", "lentils", "chickpea", "chickpeas", "bean", "beans", "peas",
		"quinoa", "cheese", "yogurt", "milk", "paneer", "peanut", "peanuts", "almond",
	}) && protein < 38 {
		return 38
	}
	return protein
}

func hasAnyLocalTerm(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func decodeIngredientKeys(raw []byte) []string {
	var out []string
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out
	}
	return out
}

func decodeStringJSONList(raw []byte) []string {
	var out []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return mapKeysSorted(mergeNormalized(out))
}

func decodePayloadTags(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var payload struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return mapKeysSorted(mergeNormalized(payload.Tags))
}

func expandBlockedIngredients(excluded, allergies []string) map[string]struct{} {
	out := normalizeSet(excluded)
	for _, excludedIngredient := range excluded {
		for _, ingredient := range allergyIngredientKeys(normalizeLocalKey(excludedIngredient)) {
			if ingredient != "" {
				out[ingredient] = struct{}{}
			}
		}
	}
	for _, allergy := range allergies {
		for _, ingredient := range allergyIngredientKeys(normalizeLocalKey(allergy)) {
			if ingredient != "" {
				out[ingredient] = struct{}{}
			}
		}
	}
	return out
}

func mapKeysSorted(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func allergyIngredientKeys(allergy string) []string {
	switch allergy {
	case "egg", "eggs", "oeufs", "oeuf":
		return []string{"egg", "eggs", "omelette", "omelet", "mayonnaise", "mayo", "lysozyme", "lecithin"}
	case "dairy", "milk", "lait":
		return []string{"milk", "cream", "cheese", "butter", "ghee", "yogurt", "yoghurt", "lactose", "brie"}
	case "soy", "soja":
		return []string{"soy", "soybean", "soybeans", "tofu", "soy sauce"}
	case "fish", "poisson":
		return []string{"fish", "tuna", "salmon", "cod"}
	case "shellfish", "seafood", "fruits de mer", "crustaceans":
		return []string{"shrimp", "shrimps", "prawn", "prawns", "crab", "lobster", "shellfish", "seafood"}
	case "gluten":
		return []string{"gluten", "wheat", "barley", "rye", "flour", "semolina", "bread", "breaded", "dough", "pasta", "noodle", "noodles", "lasagna", "pizza", "sandwich", "sandwiches", "kulcha", "pav"}
	case "wheat", "ble":
		return []string{"wheat", "flour", "semolina", "bread", "breaded", "dough", "pasta", "noodle", "noodles", "lasagna", "pizza", "sandwich", "sandwiches", "kulcha", "pav"}
	case "peanut", "peanuts", "arachides", "arachide":
		return []string{"peanut", "peanuts", "groundnut", "groundnuts"}
	case "pork", "porc":
		return []string{"pork", "ham", "bacon", "pancetta", "prosciutto"}
	case "tree nut", "tree_nut", "nuts", "fruits a coque":
		return []string{"nut", "nuts", "almond", "hazelnut", "cashew", "pistachio", "walnut"}
	case "sesame", "sesame seeds":
		return []string{"sesame", "sesame seeds", "sesame oil"}
	default:
		if allergy == "" {
			return nil
		}
		return []string{allergy}
	}
}

func mergeNormalized(groups ...[]string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, group := range groups {
		for _, item := range group {
			key := normalizeLocalKey(item)
			if key != "" {
				out[key] = struct{}{}
			}
		}
	}
	return out
}

func normalizeSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := normalizeLocalKey(item)
		if key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func normalizeConditionSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := normalizeConditionKey(item)
		if key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func normalizeConditionKey(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	value = strings.NewReplacer(" ", "_", "-", "_").Replace(value)
	value = strings.Join(strings.Fields(value), "_")
	return value
}

func normalizeLocalKey(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer("_", " ", "-", " ", "'", " ", "/", " ")
	value = replacer.Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func localSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "local_xlsx"
	}
	return source
}
