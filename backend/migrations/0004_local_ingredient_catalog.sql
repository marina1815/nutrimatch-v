-- +goose Up
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

CREATE INDEX IF NOT EXISTS idx_local_ingredients_display_name ON catalog.local_ingredients USING gin (to_tsvector('english', display_name));
CREATE INDEX IF NOT EXISTS idx_local_allergies_display_name ON catalog.local_allergies USING gin (to_tsvector('english', display_name));

GRANT SELECT, INSERT, UPDATE, DELETE ON
    catalog.local_ingredients,
    catalog.local_allergies
TO app;

-- +goose Down
DROP TABLE IF EXISTS catalog.local_allergies;
DROP TABLE IF EXISTS catalog.local_ingredients;
