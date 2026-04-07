-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    full_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    age int NOT NULL,
    sex text NOT NULL,
    weight numeric(6,2) NOT NULL,
    height numeric(6,2) NOT NULL,
    profession text NOT NULL,
    city text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS lifestyles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    activity_level text NOT NULL,
    lifestyle_type text NOT NULL,
    goal text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS preferences (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    likes jsonb NOT NULL DEFAULT '[]',
    dislikes jsonb NOT NULL DEFAULT '[]',
    meal_styles jsonb NOT NULL DEFAULT '[]',
    meals_per_day int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS constraints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
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

CREATE TABLE IF NOT EXISTS sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz NULL,
    user_agent text NOT NULL DEFAULT '',
    ip text NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash ON sessions(refresh_token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_refresh_hash;
DROP INDEX IF EXISTS idx_sessions_user_id;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS constraints;
DROP TABLE IF EXISTS preferences;
DROP TABLE IF EXISTS lifestyles;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;

