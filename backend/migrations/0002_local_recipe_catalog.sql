-- +goose Up
CREATE TABLE IF NOT EXISTS catalog.local_recipes (
    id text PRIMARY KEY,
    title text NOT NULL,
    source text NOT NULL DEFAULT 'local_xlsx',
    calories numeric(10,2) NOT NULL DEFAULT 0,
    protein numeric(10,2) NOT NULL DEFAULT 0,
    carbs numeric(10,2) NOT NULL DEFAULT 0,
    fat numeric(10,2) NOT NULL DEFAULT 0,
    sugar numeric(10,2) NOT NULL DEFAULT 0,
    sodium_mg numeric(10,2) NOT NULL DEFAULT 0,
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.local_ingredients (
    ingredient_key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'local_xlsx',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.local_allergies (
    allergy_key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'local_xlsx',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.local_recipe_ingredients (
    recipe_id text NOT NULL REFERENCES catalog.local_recipes(id) ON DELETE CASCADE,
    ingredient_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (recipe_id, ingredient_key)
);

CREATE TABLE IF NOT EXISTS catalog.local_recipe_medical_compatibility (
    recipe_id text NOT NULL REFERENCES catalog.local_recipes(id) ON DELETE CASCADE,
    condition_key text NOT NULL REFERENCES catalog.conditions(key) ON DELETE RESTRICT,
    compatible boolean NOT NULL,
    reasons jsonb NOT NULL DEFAULT '[]',
    risk_flags jsonb NOT NULL DEFAULT '[]',
    source text NOT NULL DEFAULT 'deterministic_medical_matrix_v1',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (recipe_id, condition_key)
);

CREATE TABLE IF NOT EXISTS catalog.local_ingredient_allergies (
    ingredient_key text NOT NULL,
    allergy_key text NOT NULL,
    source text NOT NULL DEFAULT 'local_xlsx',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ingredient_key, allergy_key)
);

CREATE TABLE IF NOT EXISTS catalog.local_cross_allergies (
    ingredient_key text NOT NULL,
    cross_ingredient_key text NOT NULL,
    allergy_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ingredient_key, cross_ingredient_key, allergy_key)
);

CREATE INDEX IF NOT EXISTS idx_local_recipes_title ON catalog.local_recipes USING gin (to_tsvector('english', title));
CREATE INDEX IF NOT EXISTS idx_local_ingredients_display_name ON catalog.local_ingredients USING gin (to_tsvector('english', display_name));
CREATE INDEX IF NOT EXISTS idx_local_allergies_display_name ON catalog.local_allergies USING gin (to_tsvector('english', display_name));
CREATE INDEX IF NOT EXISTS idx_local_recipe_ingredients_key ON catalog.local_recipe_ingredients(ingredient_key);
CREATE INDEX IF NOT EXISTS idx_local_recipe_medical_condition ON catalog.local_recipe_medical_compatibility(condition_key, compatible);
CREATE INDEX IF NOT EXISTS idx_local_ingredient_allergies_allergy ON catalog.local_ingredient_allergies(allergy_key);
CREATE INDEX IF NOT EXISTS idx_local_cross_allergies_cross ON catalog.local_cross_allergies(cross_ingredient_key);

GRANT SELECT, INSERT, UPDATE, DELETE ON
    catalog.local_recipes,
    catalog.local_ingredients,
    catalog.local_allergies,
    catalog.local_recipe_ingredients,
    catalog.local_recipe_medical_compatibility,
    catalog.local_ingredient_allergies,
    catalog.local_cross_allergies
TO app;

-- +goose Down
DROP TABLE IF EXISTS catalog.local_cross_allergies;
DROP TABLE IF EXISTS catalog.local_ingredient_allergies;
DROP TABLE IF EXISTS catalog.local_recipe_medical_compatibility;
DROP TABLE IF EXISTS catalog.local_recipe_ingredients;
DROP TABLE IF EXISTS catalog.local_allergies;
DROP TABLE IF EXISTS catalog.local_ingredients;
DROP TABLE IF EXISTS catalog.local_recipes;
