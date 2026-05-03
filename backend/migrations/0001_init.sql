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
    preferred_mfa_method text NOT NULL DEFAULT '' CHECK (preferred_mfa_method IN ('', 'totp', 'passkey')),
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

CREATE TABLE IF NOT EXISTS identity.totp_secrets (
    user_id uuid PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    secret_ciphertext text NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    confirmed_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS identity.webauthn_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    credential_id text NOT NULL UNIQUE,
    credential_json jsonb NOT NULL DEFAULT '{}',
    display_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NULL
);

CREATE TABLE IF NOT EXISTS identity.webauthn_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('registration', 'authentication')),
    session_data jsonb NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS identity.mfa_login_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    preferred_method text NOT NULL DEFAULT '' CHECK (preferred_method IN ('totp', 'passkey')),
    allowed_methods jsonb NOT NULL DEFAULT '[]',
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz NULL,
    user_agent_hash text NOT NULL DEFAULT '',
    ip_hash text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
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
    provider_value text NOT NULL,
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

CREATE TABLE IF NOT EXISTS catalog.diets (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog.medical_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    condition_key text NOT NULL DEFAULT '',
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
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.lifestyles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    activity_level text NOT NULL CHECK (activity_level IN ('sedentary', 'light', 'moderate', 'active')),
    lifestyle_type text NOT NULL CHECK (lifestyle_type IN ('student', 'employee', 'athlete', 'mixed', 'other')),
    goal text NOT NULL CHECK (goal IN ('weight_loss', 'muscle_gain', 'weight_maintenance', 'medical_diet', 'energy_maintenance')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.preferences (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.constraints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    has_chronic_disease boolean NOT NULL DEFAULT false,
    takes_medication boolean NOT NULL DEFAULT false,
    medications text NOT NULL DEFAULT '',
    medications_index text NOT NULL DEFAULT '',
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
    derived_recommendation_tags jsonb NOT NULL DEFAULT '[]',
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

CREATE TABLE IF NOT EXISTS recommendation.daily_sets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    nutrition_profile_id uuid NOT NULL REFERENCES health.nutrition_profiles(id) ON DELETE CASCADE,
    run_id uuid NOT NULL REFERENCES recommendation.runs(id) ON DELETE CASCADE,
    query_signature text NOT NULL,
    status text NOT NULL DEFAULT 'completed',
    selection_mode text NOT NULL DEFAULT 'deterministic_fallback',
    valid_from timestamptz NOT NULL,
    valid_until timestamptz NOT NULL,
    source_summary jsonb NOT NULL DEFAULT '{}',
    decision_summary jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendation.daily_set_meals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    set_id uuid NOT NULL REFERENCES recommendation.daily_sets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    recipe_id text NOT NULL,
    title text NOT NULL,
    final_rank int NOT NULL DEFAULT 0,
    final_score numeric(10,2) NOT NULL DEFAULT 0,
    calories numeric(10,2) NOT NULL DEFAULT 0,
    protein numeric(10,2) NOT NULL DEFAULT 0,
    carbs numeric(10,2) NOT NULL DEFAULT 0,
    fat numeric(10,2) NOT NULL DEFAULT 0,
    sugar numeric(10,2) NOT NULL DEFAULT 0,
    sodium_mg numeric(10,2) NOT NULL DEFAULT 0,
    ingredients jsonb NOT NULL DEFAULT '[]',
    ai_explanation text NOT NULL DEFAULT '',
    match_reason text NOT NULL DEFAULT '',
    nutrition_confidence text NOT NULL DEFAULT 'estimated',
    nutrition_source text NOT NULL DEFAULT 'local_catalog_nutrition_estimate',
    safety_warnings jsonb NOT NULL DEFAULT '[]',
    source_provenance jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT daily_set_meals_set_recipe_unique UNIQUE (set_id, recipe_id)
);

CREATE TABLE IF NOT EXISTS recommendation.recipe_choices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    set_id uuid NOT NULL REFERENCES recommendation.daily_sets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    profile_id uuid NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
    recipe_id text NOT NULL,
    title text NOT NULL,
    ingredients jsonb NOT NULL DEFAULT '[]',
    preparation_guide text NOT NULL DEFAULT '',
    substitutions jsonb NOT NULL DEFAULT '{}',
    ai_applied boolean NOT NULL DEFAULT false,
    ai_skipped_reason text NOT NULL DEFAULT '',
    ai_output_ignored_reason text NOT NULL DEFAULT '',
    chosen_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
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
    source text NOT NULL DEFAULT 'local_catalog',
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
    previous_hash text NOT NULL DEFAULT '',
    event_hash text NOT NULL UNIQUE,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS security.audit_policies (
    key text PRIMARY KEY,
    value text NOT NULL,
    description text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
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
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON identity.webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_user_kind ON identity.webauthn_challenges(user_id, kind, expires_at);
CREATE INDEX IF NOT EXISTS idx_mfa_login_challenges_user ON identity.mfa_login_challenges(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_ingredient_aliases_key ON catalog.ingredient_aliases(ingredient_key);
CREATE INDEX IF NOT EXISTS idx_intolerance_aliases_key ON catalog.intolerance_aliases(intolerance_key);
CREATE INDEX IF NOT EXISTS idx_condition_aliases_key ON catalog.condition_aliases(condition_key);
CREATE INDEX IF NOT EXISTS idx_medical_rules_condition_key ON catalog.medical_rules(condition_key);
CREATE INDEX IF NOT EXISTS idx_profile_preference_ingredients_kind ON health.profile_preference_ingredients(user_id, kind);
CREATE INDEX IF NOT EXISTS idx_profile_intolerances_user_id ON health.profile_intolerances(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_conditions_user_id ON health.profile_conditions(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_chronic_conditions_user_id ON health.profile_chronic_conditions(user_id);
CREATE INDEX IF NOT EXISTS idx_nutrition_profiles_user_id ON health.nutrition_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_constraints_medications_index ON health.constraints(medications_index);
CREATE INDEX IF NOT EXISTS idx_profile_snapshots_profile_id ON health.profile_snapshots(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_profile_id ON recommendation.runs(profile_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_query_signature ON recommendation.runs(query_signature);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_run_id ON recommendation.candidates(run_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_profile_recipe ON recommendation.candidates(profile_id, external_recipe_id);
CREATE INDEX IF NOT EXISTS idx_daily_sets_active ON recommendation.daily_sets(user_id, profile_id, valid_until DESC);
CREATE INDEX IF NOT EXISTS idx_daily_sets_profile_created ON recommendation.daily_sets(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_daily_set_meals_set_rank ON recommendation.daily_set_meals(set_id, final_rank ASC);
CREATE INDEX IF NOT EXISTS idx_daily_set_meals_profile_recipe ON recommendation.daily_set_meals(profile_id, recipe_id);
CREATE INDEX IF NOT EXISTS idx_recipe_choices_user_recipe_expires ON recommendation.recipe_choices(user_id, profile_id, recipe_id, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_recipe_choices_expires_at ON recommendation.recipe_choices(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_recipe_choices_one_per_set ON recommendation.recipe_choices(set_id);
CREATE INDEX IF NOT EXISTS idx_profile_embeddings_profile_id ON recommendation.profile_embeddings(profile_id);
CREATE INDEX IF NOT EXISTS idx_recipe_embeddings_recipe_id ON recommendation.recipe_embeddings(external_recipe_id);
CREATE INDEX IF NOT EXISTS idx_profile_embeddings_vector_cosine ON recommendation.profile_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_recipe_embeddings_vector_cosine ON recommendation.recipe_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON security.audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON security.audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON security.audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_hash ON security.audit_events(event_hash);
CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at ON security.audit_events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_auth_failures_email_hash ON security.auth_failures(email_hash);
CREATE INDEX IF NOT EXISTS idx_auth_failures_ip_hash ON security.auth_failures(ip_hash);

INSERT INTO catalog.intolerances (key, display_name, provider_value, source) VALUES
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
    ('digestive_sensitivity', 'Digestive sensitivity', 'system')
ON CONFLICT (key) DO NOTHING;

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
        '["sugar","brown sugar","palm sugar","molasses","glucose fructose syrup","fructose powder","jam","candies","sweetened drinks","sweetened sodas","fruit syrup"]',
        '["diabetes-risk","sugary","sweetened","high-sugar","refined-carb"]',
        '[]',
        0,
        0,
        60,
        0,
        16,
        0,
        14,
        'critical',
        'Limit high sugar and refined carbohydrate meals for diabetic profiles while preserving enough protein for satiety.'
    ),
    (
        'hypertension_sodium_control',
        'hypertension',
        '',
        '["bacon","sausage","pancetta","salted meats","anchovy","pickled vegetables","soy sauce","tamari"]',
        '["hypertension-risk","high-sodium","salty","processed-meat"]',
        '[]',
        0,
        0,
        0,
        0,
        0,
        700,
        0,
        'critical',
        'Block sodium-heavy ingredients and meals for hypertensive profiles.'
    ),
    (
        'cardiac_fat_control',
        'cardiac',
        '',
        '["fried chicken","bacon","sausage","pancetta","lard","butter","cream","offal"]',
        '["cardiac-risk","cholesterol-risk","fried","high-fat","saturated-fat","processed-meat","high-sodium"]',
        '[]',
        850,
        0,
        0,
        24,
        0,
        800,
        0,
        'high',
        'Reduce saturated fat, fried meals and sodium load for cardiac profiles.'
    ),
    (
        'renal_failure_protein_sodium_control',
        'renal_failure',
        '',
        '["anchovy","offal","liver","kidney","bouzelouf","daouara","salted meats","processed meat"]',
        '["renal-risk","high-sodium","purine-rich","potassium-rich","phosphorus-rich"]',
        '[]',
        0,
        30,
        0,
        22,
        0,
        600,
        0,
        'critical',
        'Control protein, purine, potassium/phosphorus risk markers and sodium for renal profiles.'
    ),
    (
        'hypercholesterolemia_lipid_control',
        'hypercholesterolemia',
        '',
        '["butter","cream","lard","bacon","sausage","pancetta","offal","fried chicken"]',
        '["cholesterol-risk","saturated-fat","fried","high-fat","processed-meat"]',
        '[]',
        0,
        0,
        0,
        22,
        0,
        850,
        0,
        'high',
        'Avoid meals likely to be rich in saturated fat or cholesterol for hypercholesterolemia.'
    ),
    (
        'digestive_sensitivity_gentle_control',
        'digestive_sensitivity',
        '',
        '["green chili","dried chilies","cayenne pepper","hot sauce"]',
        '["digestive-risk","very-spicy","gas-forming","fried","high-fat"]',
        '[]',
        0,
        0,
        0,
        22,
        0,
        0,
        0,
        'high',
        'Avoid spicy, gas-forming and high-fat meals for digestive sensitivity.'
    ),
    (
        'warfarin_vitamin_k_guard',
        '',
        'warfarin',
        '["spinach","kale","parsley","cabbage","broccoli","brussels sprout","collard greens"]',
        '["vitamin-k-rich"]',
        '[]',
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        'high',
        'Flag vitamin K rich ingredients when a warfarin-like treatment is declared.'
    )
ON CONFLICT (code) DO UPDATE SET
    condition_key = EXCLUDED.condition_key,
    medication_pattern = EXCLUDED.medication_pattern,
    blocked_ingredients = EXCLUDED.blocked_ingredients,
    blocked_tags = EXCLUDED.blocked_tags,
    required_tags = EXCLUDED.required_tags,
    max_calories = EXCLUDED.max_calories,
    max_protein_grams = EXCLUDED.max_protein_grams,
    max_carbs_grams = EXCLUDED.max_carbs_grams,
    max_fat_grams = EXCLUDED.max_fat_grams,
    max_sugar_grams = EXCLUDED.max_sugar_grams,
    max_sodium_mg = EXCLUDED.max_sodium_mg,
    min_protein_grams = EXCLUDED.min_protein_grams,
    severity = EXCLUDED.severity,
    rationale = EXCLUDED.rationale,
    active = true,
    updated_at = now();

INSERT INTO security.audit_policies (key, value, description) VALUES
    ('retention_days', '365', 'Minimum retention for security audit events before archival review.'),
    ('auth_failure_retention_days', '30', 'Authentication failures are retained for abuse investigation and then deleted.'),
    ('rate_limit_bucket_retention_hours', '24', 'Inactive token bucket state is short-lived and purged.'),
    ('session_retention_days', '30', 'Revoked and expired sessions are retained for short-lived security investigation.'),
    ('recommendation_trace_retention_days', '90', 'Recommendation traces are retained long enough for explainability and incident review.'),
    ('hash_chain', 'sha256_previous_hash', 'Every event is chained to the previous event hash for tamper evidence.'),
    ('pii_strategy', 'fingerprint_ip_user_agent', 'Network identifiers are fingerprinted before persistence.'),
    ('access_control', 'owner_abac_default_deny', 'All sensitive resources require authenticated owner context and explicit action/sensitivity policy.'),
    ('csrf_strategy', 'signed_double_submit_session_bound', 'Unsafe browser requests require a signed CSRF token bound to the session when authenticated.'),
    ('health_data_at_rest', 'aes_256_gcm_with_hmac_blind_indexes', 'Reversible health fields use AES-256-GCM and searchable sensitive fields use keyed blind indexes.'),
    ('rate_limit_store', 'redis_or_postgres_token_bucket', 'Rate limiting uses an atomic token bucket backed by Redis when configured or PostgreSQL fallback.'),
    ('vector_policy', 'deterministic_explainable_vectors', 'Recommendation vectors must remain deterministic, traceable, versioned and non-authoritative versus health rules.')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description, updated_at = now();

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app') THEN
        GRANT USAGE ON SCHEMA identity, catalog, health, recommendation, security TO app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA identity, catalog, health, recommendation TO app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON security.auth_failures, security.rate_limit_buckets TO app;
        GRANT SELECT ON security.audit_policies TO app;
        GRANT SELECT, INSERT ON security.audit_events TO app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP SCHEMA IF EXISTS recommendation CASCADE;
DROP SCHEMA IF EXISTS health CASCADE;
DROP SCHEMA IF EXISTS catalog CASCADE;
DROP SCHEMA IF EXISTS identity CASCADE;
DROP SCHEMA IF EXISTS security CASCADE;
