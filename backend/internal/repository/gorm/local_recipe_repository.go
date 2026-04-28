package gormrepo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/marina1815/nutrimatch/internal/localdata"
	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
)

const defaultLocalRecipeLimit = 25

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
			payload, _ := json.Marshal(map[string]any{
				"source":      recipe.Source,
				"ingredients": recipe.Ingredients,
				"seedVersion": seed.Version,
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
			`, recipe.ID, recipe.Title, localSource(recipe.Source), recipe.Nutrition.Calories, recipe.Nutrition.Protein, recipe.Nutrition.Carbs, recipe.Nutrition.Fat, recipe.Nutrition.Sugar, recipe.Nutrition.SodiumMg, string(payload)).Error; err != nil {
				return err
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

	blockedIngredients := expandBlockedIngredients(query.ExcludedIngredients, query.AllergyKeys)
	allergyKeys := normalizeSet(query.AllergyKeys)
	terms := mergeNormalized(query.QueryTerms, query.Likes)

	candidates := make([]repository.LocalRecipeCandidate, 0, len(rows))
	for _, row := range rows {
		ingredients := decodeIngredientKeys(row.IngredientsJSON)
		if localRecipeBlocked(ingredients, blockedIngredients, allergyKeys, allergyMap, crossMap) {
			continue
		}

		score := localRecipeScore(row.Title, ingredients, terms, normalizeSet(query.Likes))
		protein := calibratedLocalProtein(row.Title, ingredients, row.Protein)
		candidates = append(candidates, repository.LocalRecipeCandidate{
			ID:          row.ID,
			Title:       row.Title,
			Ingredients: ingredients,
			Calories:    row.Calories,
			Protein:     protein,
			Carbs:       row.Carbs,
			Fat:         row.Fat,
			Sugar:       row.Sugar,
			SodiumMg:    row.SodiumMg,
			Score:       score,
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

func (r *LocalRecipeRepository) SuggestIngredients(ctx context.Context, query string, limit int) ([]string, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	cleaned := normalizeLocalKey(query)
	if len(cleaned) < 2 {
		return []string{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}

	like := "%" + cleaned + "%"
	var rows []struct {
		DisplayName string `gorm:"column:display_name"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT display_name
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
	`, like, like, cleaned, cleaned+"%", limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		value := normalizeLocalKey(row.DisplayName)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
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
	IngredientsJSON []byte  `gorm:"column:ingredients_json"`
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

func localRecipeBlocked(ingredients []string, blockedIngredients, allergyKeys map[string]struct{}, allergyMap, crossMap map[string]map[string]struct{}) bool {
	for _, ingredient := range ingredients {
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

func expandBlockedIngredients(excluded, allergies []string) map[string]struct{} {
	out := normalizeSet(excluded)
	for _, allergy := range allergies {
		for _, ingredient := range allergyIngredientKeys(normalizeLocalKey(allergy)) {
			if ingredient != "" {
				out[ingredient] = struct{}{}
			}
		}
	}
	return out
}

func allergyIngredientKeys(allergy string) []string {
	switch allergy {
	case "egg", "eggs", "oeufs", "oeuf":
		return []string{"egg", "eggs"}
	case "dairy", "milk", "lait":
		return []string{"milk", "cream", "cheese", "butter", "yogurt", "lactose"}
	case "soy", "soja":
		return []string{"soy", "soybean", "soybeans", "tofu", "soy sauce"}
	case "fish", "poisson":
		return []string{"fish", "tuna", "salmon", "cod"}
	case "shellfish", "seafood", "fruits de mer", "crustaceans":
		return []string{"shrimp", "crab", "lobster", "shellfish", "seafood"}
	case "gluten":
		return []string{"gluten", "wheat", "barley", "rye", "flour", "semolina"}
	case "wheat", "ble":
		return []string{"wheat", "flour", "semolina"}
	case "peanut", "peanuts", "arachides", "arachide":
		return []string{"peanut", "peanuts"}
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
