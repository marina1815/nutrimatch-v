-- +goose Up
GRANT SELECT, INSERT, UPDATE, DELETE ON
    catalog.local_recipes,
    catalog.local_ingredients,
    catalog.local_allergies,
    catalog.local_recipe_ingredients,
    catalog.local_ingredient_allergies,
    catalog.local_cross_allergies
TO app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE, DELETE ON
    catalog.local_recipes,
    catalog.local_ingredients,
    catalog.local_allergies,
    catalog.local_recipe_ingredients,
    catalog.local_ingredient_allergies,
    catalog.local_cross_allergies
FROM app;
