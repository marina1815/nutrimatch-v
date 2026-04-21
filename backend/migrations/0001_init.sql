-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS health;
CREATE SCHEMA IF NOT EXISTS recommendation;
CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS identity.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    full_name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    email_verified boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS identity.external_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text NOT NULL DEFAULT '',
    email_verified boolean NOT NULL DEFAULT false,
    last_login_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT external_identities_provider_subject_unique UNIQUE (provider, issuer, subject)
);

CREATE TABLE IF NOT EXISTS identity.sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    auth_method text NOT NULL DEFAULT 'local',
    refresh_token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz NULL,
    user_agent_hash text NOT NULL DEFAULT '',
    ip_hash text NOT NULL DEFAULT '',
    csrf_binding_id uuid NOT NULL DEFAULT gen_random_uuid(),
    CHECK (idle_expires_at <= expires_at)
);

CREATE TABLE IF NOT EXISTS catalog.ingredients (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'user',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.ingredient_aliases (
    alias text PRIMARY KEY,
    ingredient_key text NOT NULL REFERENCES catalog.ingredients(key) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'und',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.intolerances (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    spoonacular_value text NOT NULL,
    source text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.intolerance_aliases (
    alias text PRIMARY KEY,
    intolerance_key text NOT NULL REFERENCES catalog.intolerances(key) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'und',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.conditions (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.condition_aliases (
    alias text PRIMARY KEY,
    condition_key text NOT NULL REFERENCES catalog.conditions(key) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'und',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.meal_styles (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.meal_types (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.diets (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.cuisines (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    source text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.spoonacular_param_map (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain text NOT NULL,
    internal_key text NOT NULL,
    spoonacular_param text NOT NULL,
    spoonacular_value text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT spoonacular_param_map_unique UNIQUE (domain, internal_key, spoonacular_param)
);

CREATE TABLE IF NOT EXISTS catalog.medical_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    condition_key text NOT NULL DEFAULT '' REFERENCES catalog.conditions(key) ON DELETE RESTRICT,
    medication_pattern text NOT NULL DEFAULT '',
    blocked_ingredients jsonb NOT NULL DEFAULT '[]',
    blocked_tags jsonb NOT NULL DEFAULT '[]',
    required_tags jsonb NOT NULL DEFAULT '[]',
    max_calories numeric(10,2) NOT NULL DEFAULT 0,
    max_protein_grams numeric(10,2) NOT NULL DEFAULT 0,
    max_carbs_grams numeric(10,2) NOT NULL DEFAULT 0,
    max_fat_grams numeric(10,2) NOT NULL DEFAULT 0,
    max_sugar_grams numeric(10,2) NOT NULL DEFAULT 0,
    max_sodium_mg numeric(10,2) NOT NULL DEFAULT 0,
    min_protein_grams numeric(10,2) NOT NULL DEFAULT 0,
    severity text NOT NULL DEFAULT 'high' CHECK (severity IN ('medium', 'high', 'critical')),
    rationale text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    age int NOT NULL CHECK (age BETWEEN 10 AND 120),
    sex text NOT NULL CHECK (sex IN ('male', 'female')),
    weight numeric(6,2) NOT NULL CHECK (weight BETWEEN 20 AND 400),
    height numeric(6,2) NOT NULL CHECK (height BETWEEN 80 AND 250),
    profession text NOT NULL,
    city text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.lifestyles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    activity_level text NOT NULL CHECK (activity_level IN ('sedentary', 'light', 'moderate', 'active')),
    lifestyle_type text NOT NULL CHECK (lifestyle_type IN ('student', 'employee', 'athlete', 'mixed', 'other')),
    goal text NOT NULL CHECK (goal IN ('weight_loss', 'muscle_gain', 'weight_maintenance', 'medical_diet', 'energy_maintenance')),
    max_ready_time int NOT NULL DEFAULT 45 CHECK (max_ready_time BETWEEN 5 AND 240),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.preferences (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    meals_per_day int NOT NULL CHECK (meals_per_day BETWEEN 1 AND 8),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.constraints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    has_chronic_disease boolean NOT NULL DEFAULT false,
    takes_medication boolean NOT NULL DEFAULT false,
    medications text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.profile_preference_ingredients (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    ingredient_key text NOT NULL REFERENCES catalog.ingredients(key) ON DELETE RESTRICT,
    kind text NOT NULL CHECK (kind IN ('like', 'dislike', 'exclude')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, ingredient_key, kind)
);

CREATE TABLE IF NOT EXISTS health.profile_meal_styles (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    meal_style_key text NOT NULL REFERENCES catalog.meal_styles(key) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, meal_style_key)
);

CREATE TABLE IF NOT EXISTS health.profile_meal_types (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    meal_type_key text NOT NULL REFERENCES catalog.meal_types(key) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, meal_type_key)
);

CREATE TABLE IF NOT EXISTS health.profile_cuisines (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    cuisine_key text NOT NULL REFERENCES catalog.cuisines(key) ON DELETE RESTRICT,
    kind text NOT NULL CHECK (kind IN ('preferred', 'excluded')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, cuisine_key, kind)
);

CREATE TABLE IF NOT EXISTS health.profile_intolerances (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    intolerance_key text NOT NULL REFERENCES catalog.intolerances(key) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, intolerance_key)
);

CREATE TABLE IF NOT EXISTS health.profile_conditions (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    condition_key text NOT NULL REFERENCES catalog.conditions(key) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, condition_key)
);

CREATE TABLE IF NOT EXISTS health.profile_chronic_conditions (
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    condition_key text NOT NULL REFERENCES catalog.conditions(key) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, condition_key)
);

CREATE TABLE IF NOT EXISTS health.nutrition_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    bmi numeric(8,2) NOT NULL,
    bmi_category text NOT NULL,
    bmr numeric(10,2) NOT NULL,
    estimated_calories numeric(10,2) NOT NULL,
    target_calories numeric(10,2) NOT NULL,
    target_protein_grams numeric(10,2) NOT NULL,
    target_carbs_grams numeric(10,2) NOT NULL,
    target_fat_grams numeric(10,2) NOT NULL,
    max_meal_calories numeric(10,2) NOT NULL,
    min_protein_per_meal numeric(10,2) NOT NULL,
    max_carbs_per_meal numeric(10,2) NOT NULL,
    max_fat_per_meal numeric(10,2) NOT NULL,
    max_sugar_per_meal numeric(10,2) NOT NULL,
    max_sodium_mg_per_meal numeric(10,2) NOT NULL,
    derived_restrictions jsonb NOT NULL DEFAULT '[]',
    derived_excluded jsonb NOT NULL DEFAULT '[]',
    recommended_meal_styles jsonb NOT NULL DEFAULT '[]',
    metadata jsonb NOT NULL DEFAULT '{}',
    calculated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.profile_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendation.runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    nutrition_profile_id uuid NOT NULL REFERENCES health.nutrition_profiles(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'completed',
    query_signature text NOT NULL,
    source_summary jsonb NOT NULL DEFAULT '{}',
    decision_summary jsonb NOT NULL DEFAULT '{}',
    external_trace jsonb NOT NULL DEFAULT '{}',
    correlated_request_id text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendation.candidates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id uuid NOT NULL REFERENCES recommendation.runs(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    external_recipe_id text NOT NULL,
    title text NOT NULL,
    source text NOT NULL,
    stage text NOT NULL,
    accepted boolean NOT NULL DEFAULT false,
    final_rank int NOT NULL DEFAULT 0,
    final_score numeric(10,2) NOT NULL DEFAULT 0,
    calories numeric(10,2) NOT NULL DEFAULT 0,
    protein numeric(10,2) NOT NULL DEFAULT 0,
    carbs numeric(10,2) NOT NULL DEFAULT 0,
    fat numeric(10,2) NOT NULL DEFAULT 0,
    sugar numeric(10,2) NOT NULL DEFAULT 0,
    sodium_mg numeric(10,2) NOT NULL DEFAULT 0,
    ingredients jsonb NOT NULL DEFAULT '[]',
    tags jsonb NOT NULL DEFAULT '[]',
    accepted_reasons jsonb NOT NULL DEFAULT '[]',
    rejected_reasons jsonb NOT NULL DEFAULT '[]',
    score_breakdown jsonb NOT NULL DEFAULT '{}',
    filter_decisions jsonb NOT NULL DEFAULT '{}',
    source_provenance jsonb NOT NULL DEFAULT '{}',
    explanation text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendation.external_recipe_cache (
    external_recipe_id text PRIMARY KEY,
    source text NOT NULL DEFAULT 'spoonacular',
    payload jsonb NOT NULL DEFAULT '{}',
    fetched_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS recommendation.search_response_cache (
    cache_key text PRIMARY KEY,
    source text NOT NULL DEFAULT 'spoonacular',
    payload jsonb NOT NULL DEFAULT '{}',
    fetched_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS recommendation.ingredient_resolution_cache (
    normalized_query text PRIMARY KEY,
    ingredient_key text NOT NULL REFERENCES catalog.ingredients(key) ON DELETE RESTRICT,
    source text NOT NULL DEFAULT 'spoonacular',
    confidence numeric(5,4) NOT NULL DEFAULT 0,
    payload jsonb NOT NULL DEFAULT '{}',
    fetched_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS recommendation.profile_embeddings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    embedding_version text NOT NULL,
    source_hash text NOT NULL,
    embedding vector(768),
    metadata jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT profile_embeddings_unique UNIQUE (profile_id, embedding_version)
);

CREATE TABLE IF NOT EXISTS recommendation.recipe_embeddings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    external_recipe_id text NOT NULL,
    source text NOT NULL DEFAULT 'spoonacular',
    embedding_version text NOT NULL,
    source_hash text NOT NULL,
    embedding vector(768),
    metadata jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT recipe_embeddings_unique UNIQUE (external_recipe_id, source, embedding_version)
);

CREATE TABLE IF NOT EXISTS security.audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    session_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    event_type text NOT NULL,
    resource_type text NOT NULL,
    resource_id text NOT NULL DEFAULT '',
    outcome text NOT NULL,
    ip text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    request_id text NOT NULL DEFAULT '',
    details jsonb NOT NULL DEFAULT '{}',
    external_trace jsonb NOT NULL DEFAULT '{}',
    occurred_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS security.auth_failures (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email_hash text NOT NULL DEFAULT '',
    ip_hash text NOT NULL DEFAULT '',
    reason text NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS security.rate_limit_buckets (
    key text PRIMARY KEY,
    bucket_type text NOT NULL,
    tokens numeric(12,4) NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON identity.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash ON identity.sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_idle_expires_at ON identity.sessions(idle_expires_at);
CREATE INDEX IF NOT EXISTS idx_external_identities_user_id ON identity.external_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_ingredient_aliases_key ON catalog.ingredient_aliases(ingredient_key);
CREATE INDEX IF NOT EXISTS idx_intolerance_aliases_key ON catalog.intolerance_aliases(intolerance_key);
CREATE INDEX IF NOT EXISTS idx_condition_aliases_key ON catalog.condition_aliases(condition_key);
CREATE INDEX IF NOT EXISTS idx_medical_rules_condition_key ON catalog.medical_rules(condition_key);
CREATE INDEX IF NOT EXISTS idx_profile_preference_ingredients_kind ON health.profile_preference_ingredients(user_id, kind);
CREATE INDEX IF NOT EXISTS idx_profile_meal_styles_user_id ON health.profile_meal_styles(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_intolerances_user_id ON health.profile_intolerances(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_conditions_user_id ON health.profile_conditions(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_chronic_conditions_user_id ON health.profile_chronic_conditions(user_id);
CREATE INDEX IF NOT EXISTS idx_nutrition_profiles_user_id ON health.nutrition_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_snapshots_profile_id ON health.profile_snapshots(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_profile_id ON recommendation.runs(profile_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_query_signature ON recommendation.runs(query_signature);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_run_id ON recommendation.candidates(run_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_profile_recipe ON recommendation.candidates(profile_id, external_recipe_id);
CREATE INDEX IF NOT EXISTS idx_external_recipe_cache_expires_at ON recommendation.external_recipe_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_search_response_cache_expires_at ON recommendation.search_response_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_ingredient_resolution_cache_expires_at ON recommendation.ingredient_resolution_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_profile_embeddings_profile_id ON recommendation.profile_embeddings(profile_id);
CREATE INDEX IF NOT EXISTS idx_recipe_embeddings_recipe_id ON recommendation.recipe_embeddings(external_recipe_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON security.audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON security.audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON security.audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_auth_failures_email_hash ON security.auth_failures(email_hash);
CREATE INDEX IF NOT EXISTS idx_auth_failures_ip_hash ON security.auth_failures(ip_hash);

INSERT INTO catalog.meal_styles (key, display_name, source) VALUES
    ('traditional', 'Traditional', 'system'),
    ('healthy', 'Healthy', 'system'),
    ('middle eastern', 'Middle Eastern', 'system'),
    ('modern', 'Modern', 'system'),
    ('cold', 'Cold Meals', 'system'),
    ('quick', 'Quick', 'system'),
    ('balanced', 'Balanced', 'system'),
    ('high-protein', 'High Protein', 'system'),
    ('low-sodium', 'Low Sodium', 'system'),
    ('low-sugar', 'Low Sugar', 'system')
ON CONFLICT (key) DO NOTHING;

INSERT INTO catalog.intolerances (key, display_name, spoonacular_value, source) VALUES
    ('dairy', 'Dairy', 'dairy', 'system'),
    ('egg', 'Egg', 'egg', 'system'),
    ('gluten', 'Gluten', 'gluten', 'system'),
    ('grain', 'Grain', 'grain', 'system'),
    ('peanut', 'Peanut', 'peanut', 'system'),
    ('seafood', 'Seafood', 'seafood', 'system'),
    ('sesame', 'Sesame', 'sesame', 'system'),
    ('shellfish', 'Shellfish', 'shellfish', 'system'),
    ('soy', 'Soy', 'soy', 'system'),
    ('sulfite', 'Sulfite', 'sulfite', 'system'),
    ('tree_nut', 'Tree Nut', 'tree nut', 'system'),
    ('wheat', 'Wheat', 'wheat', 'system')
ON CONFLICT (key) DO NOTHING;

INSERT INTO catalog.conditions (key, display_name, source) VALUES
    ('diabetes', 'Diabetes', 'system'),
    ('hypertension', 'Hypertension', 'system'),
    ('cardiac', 'Cardiac disease', 'system'),
    ('renal_failure', 'Renal failure', 'system'),
    ('hypercholesterolemia', 'Hypercholesterolemia', 'system'),
    ('digestive_sensitivity', 'Digestive sensitivity', 'system'),
    ('other', 'Other', 'system')
ON CONFLICT (key) DO NOTHING;

INSERT INTO catalog.spoonacular_param_map (domain, internal_key, spoonacular_param, spoonacular_value) VALUES
    ('intolerance', 'tree_nut', 'intolerances', 'tree nut'),
    ('intolerance', 'shellfish', 'intolerances', 'shellfish'),
    ('intolerance', 'dairy', 'intolerances', 'dairy'),
    ('intolerance', 'egg', 'intolerances', 'egg'),
    ('intolerance', 'gluten', 'intolerances', 'gluten'),
    ('intolerance', 'grain', 'intolerances', 'grain'),
    ('intolerance', 'peanut', 'intolerances', 'peanut'),
    ('intolerance', 'seafood', 'intolerances', 'seafood'),
    ('intolerance', 'sesame', 'intolerances', 'sesame'),
    ('intolerance', 'soy', 'intolerances', 'soy'),
    ('intolerance', 'sulfite', 'intolerances', 'sulfite'),
    ('intolerance', 'wheat', 'intolerances', 'wheat')
ON CONFLICT (domain, internal_key, spoonacular_param) DO NOTHING;

INSERT INTO catalog.medical_rules (
    code,
    condition_key,
    medication_pattern,
    blocked_ingredients,
    blocked_tags,
    required_tags,
    max_calories,
    max_protein_grams,
    max_carbs_grams,
    max_fat_grams,
    max_sugar_grams,
    max_sodium_mg,
    min_protein_grams,
    severity,
    rationale
) VALUES
    (
        'diabetes_sugar_control',
        'diabetes',
        '',
        '[]',
        '["sugary","dessert"]',
        '["high-protein"]',
        0,
        0,
        60,
        0,
        18,
        0,
        18,
        'critical',
        'Limit fast sugar loads and preserve a minimum protein floor.'
    ),
    (
        'hypertension_sodium_control',
        'hypertension',
        '',
        '["bacon","sausage"]',
        '["salty"]',
        '["low-sodium"]',
        0,
        0,
        0,
        0,
        0,
        700,
        0,
        'critical',
        'Strong sodium restriction for hypertensive profiles.'
    ),
    (
        'cardiac_fat_control',
        'cardiac',
        '',
        '["fried chicken"]',
        '["fried"]',
        '[]',
        850,
        0,
        0,
        24,
        0,
        800,
        0,
        'high',
        'Reduce fat and sodium load for cardiac profiles.'
    ),
    (
        'renal_failure_protein_sodium_control',
        'renal_failure',
        '',
        '["anchovy"]',
        '[]',
        '[]',
        0,
        28,
        0,
        18,
        0,
        600,
        0,
        'critical',
        'Control high protein, fat and sodium meals for renal profiles.'
    ),
    (
        'warfarin_grapefruit_guard',
        '',
        'warfarin',
        '["grapefruit"]',
        '[]',
        '[]',
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'high',
        'Block ingredients declared as problematic for the medication pattern.'
    ),
    (
        'statin_grapefruit_guard',
        '',
        'statin',
        '["grapefruit"]',
        '[]',
        '[]',
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'high',
        'Avoid grapefruit when a statin-like treatment is declared.'
    )
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS security.rate_limit_buckets;
DROP TABLE IF EXISTS security.auth_failures;
DROP TABLE IF EXISTS security.audit_events;
DROP TABLE IF EXISTS recommendation.recipe_embeddings;
DROP TABLE IF EXISTS recommendation.profile_embeddings;
DROP TABLE IF EXISTS recommendation.ingredient_resolution_cache;
DROP TABLE IF EXISTS recommendation.search_response_cache;
DROP TABLE IF EXISTS recommendation.external_recipe_cache;
DROP TABLE IF EXISTS recommendation.candidates;
DROP TABLE IF EXISTS recommendation.runs;
DROP TABLE IF EXISTS health.profile_snapshots;
DROP TABLE IF EXISTS health.nutrition_profiles;
DROP TABLE IF EXISTS health.profile_chronic_conditions;
DROP TABLE IF EXISTS health.profile_conditions;
DROP TABLE IF EXISTS health.profile_intolerances;
DROP TABLE IF EXISTS health.profile_cuisines;
DROP TABLE IF EXISTS health.profile_meal_types;
DROP TABLE IF EXISTS health.profile_meal_styles;
DROP TABLE IF EXISTS health.profile_preference_ingredients;
DROP TABLE IF EXISTS health.constraints;
DROP TABLE IF EXISTS health.preferences;
DROP TABLE IF EXISTS health.lifestyles;
DROP TABLE IF EXISTS health.profiles;
DROP TABLE IF EXISTS catalog.medical_rules;
DROP TABLE IF EXISTS catalog.spoonacular_param_map;
DROP TABLE IF EXISTS catalog.cuisines;
DROP TABLE IF EXISTS catalog.diets;
DROP TABLE IF EXISTS catalog.meal_types;
DROP TABLE IF EXISTS catalog.meal_styles;
DROP TABLE IF EXISTS catalog.condition_aliases;
DROP TABLE IF EXISTS catalog.conditions;
DROP TABLE IF EXISTS catalog.intolerance_aliases;
DROP TABLE IF EXISTS catalog.intolerances;
DROP TABLE IF EXISTS catalog.ingredient_aliases;
DROP TABLE IF EXISTS catalog.ingredients;
DROP TABLE IF EXISTS identity.sessions;
DROP TABLE IF EXISTS identity.external_identities;
DROP TABLE IF EXISTS identity.users;

DROP SCHEMA IF EXISTS security;
DROP SCHEMA IF EXISTS recommendation;
DROP SCHEMA IF EXISTS health;
DROP SCHEMA IF EXISTS catalog;
DROP SCHEMA IF EXISTS identity;
