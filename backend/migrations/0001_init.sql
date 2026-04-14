-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS health;
CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS identity.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    full_name text NOT NULL,
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
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz NULL,
    user_agent text NOT NULL DEFAULT '',
    ip text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS health.profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    age int NOT NULL,
    sex text NOT NULL,
    weight numeric(6,2) NOT NULL,
    height numeric(6,2) NOT NULL,
    profession text NOT NULL,
    city text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.lifestyles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    activity_level text NOT NULL,
    lifestyle_type text NOT NULL,
    goal text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.preferences (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    likes jsonb NOT NULL DEFAULT '[]',
    dislikes jsonb NOT NULL DEFAULT '[]',
    meal_styles jsonb NOT NULL DEFAULT '[]',
    meals_per_day int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.constraints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES identity.users(id) ON DELETE CASCADE,
    allergies jsonb NOT NULL DEFAULT '[]',
    conditions jsonb NOT NULL DEFAULT '[]',
    excluded_ingredients jsonb NOT NULL DEFAULT '[]',
    has_chronic_disease boolean NOT NULL DEFAULT false,
    chronic_diseases jsonb NOT NULL DEFAULT '[]',
    takes_medication boolean NOT NULL DEFAULT false,
    medications text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
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

CREATE TABLE IF NOT EXISTS health.medical_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    condition_key text NOT NULL,
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
    severity text NOT NULL DEFAULT 'high',
    rationale text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health.recommendation_runs (
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

CREATE TABLE IF NOT EXISTS health.recommendation_candidates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id uuid NOT NULL REFERENCES health.recommendation_runs(id) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON identity.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash ON identity.sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_external_identities_user_id ON identity.external_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_nutrition_profiles_user_id ON health.nutrition_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_medical_rules_condition_key ON health.medical_rules(condition_key);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_profile_id ON health.recommendation_runs(profile_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_runs_query_signature ON health.recommendation_runs(query_signature);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_run_id ON health.recommendation_candidates(run_id);
CREATE INDEX IF NOT EXISTS idx_recommendation_candidates_profile_recipe ON health.recommendation_candidates(profile_id, external_recipe_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON security.audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON security.audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON security.audit_events(request_id);

INSERT INTO health.medical_rules (
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
        'Limite les repas trop riches en sucres rapides et favorise un apport proteique minimal.'
    ),
    (
        'hypertension_sodium_control',
        'hypertension',
        '',
        '["bacon","sausage"]',
        '["salty"]',
        '[]',
        0,
        0,
        0,
        0,
        0,
        700,
        0,
        'critical',
        'Limite fortement le sodium par repas pour les profils avec hypertension.'
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
        'Reduit la charge lipidique et sodique des repas pour les profils cardiaques.'
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
        'Controle les repas trop riches en proteines, graisses et sodium pour les profils renaux.'
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
        'Bloque les ingredients susceptibles de poser probleme en presence d un traitement declare.'
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
        'Bloque le pamplemousse pour les traitements declares de type statine.'
    )
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP INDEX IF EXISTS security.idx_audit_events_request_id;
DROP INDEX IF EXISTS security.idx_audit_events_event_type;
DROP INDEX IF EXISTS security.idx_audit_events_user_id;
DROP INDEX IF EXISTS health.idx_recommendation_candidates_profile_recipe;
DROP INDEX IF EXISTS health.idx_recommendation_candidates_run_id;
DROP INDEX IF EXISTS health.idx_recommendation_runs_query_signature;
DROP INDEX IF EXISTS health.idx_recommendation_runs_profile_id;
DROP INDEX IF EXISTS health.idx_medical_rules_condition_key;
DROP INDEX IF EXISTS health.idx_nutrition_profiles_user_id;
DROP INDEX IF EXISTS identity.idx_external_identities_user_id;
DROP INDEX IF EXISTS identity.idx_sessions_refresh_hash;
DROP INDEX IF EXISTS identity.idx_sessions_user_id;

DROP TABLE IF EXISTS security.audit_events;
DROP TABLE IF EXISTS health.recommendation_candidates;
DROP TABLE IF EXISTS health.recommendation_runs;
DROP TABLE IF EXISTS health.medical_rules;
DROP TABLE IF EXISTS health.nutrition_profiles;
DROP TABLE IF EXISTS health.constraints;
DROP TABLE IF EXISTS health.preferences;
DROP TABLE IF EXISTS health.lifestyles;
DROP TABLE IF EXISTS health.profiles;
DROP TABLE IF EXISTS identity.sessions;
DROP TABLE IF EXISTS identity.external_identities;
DROP TABLE IF EXISTS identity.users;

DROP SCHEMA IF EXISTS security;
DROP SCHEMA IF EXISTS health;
DROP SCHEMA IF EXISTS identity;

