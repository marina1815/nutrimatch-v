-- +goose Up
-- Intentionally left as a no-op.
-- Local ingredient and allergy catalog tables are already created in 0002_local_recipe_catalog.sql.

-- +goose Down
-- Intentionally left as a no-op.
-- The owning migration 0002_local_recipe_catalog.sql is responsible for dropping the catalog objects.
