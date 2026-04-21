# Spoonacular Contract

This document freezes the phase-1 contract between:

- the user-facing profile form
- NutriMatch canonical business vocabulary
- Spoonacular query parameters
- deterministic backend health filters

Goal: remove ambiguity before the schema and API refactor.

## 1. Scope

NutriMatch will use Spoonacular as:

- ingredient resolver
- recipe search provider
- recipe nutrition provider

NutriMatch will not delegate medical safety decisions to Spoonacular.
Medical logic stays internal and deterministic.

## 2. Official references

Primary references used for this contract:

- Spoonacular Food API home: `https://spoonacular.com/food-api`
- Spoonacular official Postman collection: `https://www.postman.com/spoonacular-api/spoonacular-api/collection/7431899-ef0368a7-643c-4c87-975c-68399d4f0c12`
- Recipes `complexSearch`: `https://api.spoonacular.com/recipes/complexSearch`
- Ingredient search: `https://api.spoonacular.com/food/ingredients/search`
- Ingredient autocomplete: `https://api.spoonacular.com/food/ingredients/autocomplete`
- Recipe information: `https://api.spoonacular.com/recipes/{id}/information`

## 3. Design rules

1. The frontend must not invent English tokens on its own.
2. The frontend sends canonical keys or free-text ingredient drafts.
3. The backend owns all normalization and Spoonacular translation.
4. Free-text ingredients are resolved server-side against Spoonacular.
5. Health conditions are internal concepts, not direct Spoonacular filters.
6. Allergies and intolerances use canonical values aligned with Spoonacular where possible.
7. AI can re-rank only already approved candidates.

## 4. Canonical profile model

The target canonical profile separates concerns that are currently mixed in the form.

### 4.1 Identity and physiology

- full name
- age
- sex
- weight kg
- height cm
- profession
- city

### 4.2 Lifestyle

- activity level
- lifestyle type
- goal
- meals per day
- max ready time (new)

### 4.3 Explicit food preferences

- liked ingredient ids or canonical names
- disliked ingredient ids or canonical names
- excluded ingredient ids or canonical names
- preferred cuisines
- excluded cuisines
- preferred meal types
- preferred dietary styles

### 4.4 Health constraints

- intolerances
- chronic conditions
- medications
- chronic disease flag
- medication flag

## 5. Canonical values

## 5.1 Goals

Canonical keys:

- `weight_loss`
- `muscle_gain`
- `weight_maintenance`
- `medical_diet`
- `energy_maintenance`

These stay internal and drive deterministic thresholds.

## 5.2 Activity levels

Canonical keys:

- `sedentary`
- `light`
- `moderate`
- `active`

## 5.3 Intolerances

Canonical keys aligned to Spoonacular-compatible values:

- `dairy`
- `egg`
- `gluten`
- `grain`
- `peanut`
- `seafood`
- `sesame`
- `shellfish`
- `soy`
- `sulfite`
- `tree_nut`
- `wheat`

Storage rule:

- persist canonical keys
- convert `tree_nut` to Spoonacular `tree nut`
- convert all other keys to their Spoonacular token directly

## 5.4 Health conditions

Canonical internal keys:

- `diabetes`
- `hypertension`
- `cardiac`
- `renal_failure`
- `hypercholesterolemia`
- `digestive_sensitivity`
- `other`

These do not map directly to Spoonacular.
They map to deterministic internal medical rules and nutrient limits.

## 5.5 Meal types

Canonical keys aligned to Spoonacular `type` values where possible:

- `main_course`
- `side_dish`
- `breakfast`
- `lunch`
- `dinner`
- `snack`
- `salad`
- `soup`
- `dessert`
- `appetizer`
- `beverage`

Translation rule:

- `main_course` -> `main course`
- `side_dish` -> `side dish`
- all others become their space-separated Spoonacular value if needed

## 5.6 Dietary styles

Canonical keys:

- `healthy`
- `balanced`
- `quick`
- `cold`
- `traditional`
- `modern`
- `high-protein`
- `low-sodium`
- `low-sugar`

These are internal ranking or query helpers.
They do not always map 1:1 to a Spoonacular parameter.

## 5.7 Cuisines

Canonical keys should match Spoonacular cuisine families whenever available.
Initial supported values:

- `african`
- `american`
- `asian`
- `mediterranean`
- `middle_eastern`
- `european`
- `mexican`

The exact supported list can expand later.

## 6. Ingredient handling

Ingredients are too open-ended to hard-code exhaustively.
NutriMatch will support them through server-side resolution.

### 6.1 Input strategy

The user may type:

- chicken
- riz
- oeufs
- lait
- crevettes
- pamplemousse

The frontend stores drafts only.
The backend resolves each draft into:

- normalized input
- canonical English display token
- optional Spoonacular ingredient id
- confidence score
- alias source

### 6.2 Resolution strategy

Resolution order:

1. exact canonical alias map
2. known French synonym map
3. internal synonym table
4. Spoonacular ingredient autocomplete/search
5. manual fallback as normalized plain text if confidence is low

### 6.3 Storage strategy

Store both:

- user-entered raw value
- canonical resolved value

Never use raw free text directly in safety rules if canonical resolution failed.

## 7. Mapping to Spoonacular

## 7.1 Recipe search endpoint

Main endpoint:

- `GET /recipes/complexSearch`

Primary parameters NutriMatch will use:

- `query`
- `cuisine`
- `excludeCuisine`
- `diet`
- `intolerances`
- `includeIngredients`
- `excludeIngredients`
- `type`
- `maxReadyTime`
- `addRecipeInformation=true`
- `addRecipeNutrition=true`
- `fillIngredients=true`
- `number`
- nutrition bounds such as:
  - `minProtein`
  - `maxProtein`
  - `minCalories`
  - `maxCalories`
  - `maxCarbs`
  - `maxFat`
  - `maxSodium`
  - `maxSugar`

## 7.2 Internal to Spoonacular conversion

- canonical intolerances -> `intolerances`
- preferred cuisines -> `cuisine`
- excluded cuisines -> `excludeCuisine`
- preferred meal type -> `type`
- liked ingredients -> `includeIngredients`
- disliked ingredients -> `excludeIngredients`
- excluded ingredients -> `excludeIngredients`
- nutrition profile thresholds -> `min/max` nutrient params

## 7.3 Query composition rules

1. Safety filters first.
2. Internal medical exclusions are merged into `excludeIngredients`.
3. Explicit user exclusions always override preferences.
4. `query` must be assistive only, never authoritative for safety.
5. If a medical rule and Spoonacular disagree, internal deterministic logic wins.

## 8. Health logic boundaries

Spoonacular may help with:

- recipe candidates
- structured ingredient lists
- nutrition facts
- broad intolerance filtering

Spoonacular must not decide:

- whether a profile is medically sensitive
- whether a medication interaction is acceptable
- whether a candidate bypasses a chronic disease rule

These stay in NutriMatch deterministic rules.

## 9. Form changes required

The current form mixes several concepts.
The target form contract should split:

- allergies/intolerances
- chronic conditions
- disliked ingredients
- excluded ingredients
- cuisines
- meal types
- dietary styles

New form data to add:

- max ready time
- preferred cuisines
- excluded cuisines
- preferred meal types

Current field `mealStyles` should eventually be decomposed.

## 10. Immediate implementation consequences

1. Normalize French and English values in backend before persistence.
2. Persist canonical condition and intolerance keys.
3. Resolve free-text ingredients server-side.
4. Stop relying on raw French labels in recommendation logic.
5. Move frontend constants toward canonical keys and display labels.
