package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	AppPort  string
	DBURL    string
	BodyLimitBytes int64

	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	RefreshTokenPepper string

	Argon2Time       uint32
	Argon2Memory     uint32
	Argon2Threads    uint8
	Argon2KeyLength  uint32
	Argon2SaltLength uint32

	CookieNameRefresh string
	CookieDomain      string
	CookieSecure      bool
	CookieSameSite    string

	CORSOrigins string

	NutritionAPIBaseURL string
	NutritionAPIKey     string
	AIAPIBaseURL        string
	AIAPIKey            string

	SpoonacularBaseURL string
	SpoonacularAPIKey  string
	GoogleAIBaseURL    string
	GoogleAIAPIKey     string
	GoogleAIModel      string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		DBURL:   getEnv("DATABASE_URL", ""),
		BodyLimitBytes: getEnvInt64("BODY_LIMIT_BYTES", 1048576),

		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTIssuer:   getEnv("JWT_ISSUER", "nutrimatch"),
		JWTAudience: getEnv("JWT_AUDIENCE", "nutrimatch_users"),
		AccessTokenTTL:  time.Duration(getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute,
		RefreshTokenTTL: time.Duration(getEnvInt("REFRESH_TOKEN_TTL_HOURS", 720)) * time.Hour,
		RefreshTokenPepper: getEnv("REFRESH_TOKEN_PEPPER", getEnv("JWT_SECRET", "")),

		Argon2Time:       uint32(getEnvInt("ARGON2_TIME", 1)),
		Argon2Memory:     uint32(getEnvInt("ARGON2_MEMORY", 65536)),
		Argon2Threads:    uint8(getEnvInt("ARGON2_THREADS", 4)),
		Argon2KeyLength:  uint32(getEnvInt("ARGON2_KEY_LENGTH", 32)),
		Argon2SaltLength: uint32(getEnvInt("ARGON2_SALT_LENGTH", 16)),

		CookieNameRefresh: getEnv("COOKIE_NAME_REFRESH", "nm_refresh"),
		CookieDomain:      getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:      getEnvBool("COOKIE_SECURE", false),
		CookieSameSite:    getEnv("COOKIE_SAMESITE", "Lax"),

		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:3000"),

		NutritionAPIBaseURL: getEnv("NUTRITION_API_BASE_URL", ""),
		NutritionAPIKey:     getEnv("NUTRITION_API_KEY", ""),
		AIAPIBaseURL:        getEnv("AI_API_BASE_URL", ""),
		AIAPIKey:            getEnv("AI_API_KEY", ""),

		SpoonacularBaseURL: getEnv("SPOONACULAR_BASE_URL", "https://api.spoonacular.com"),
		SpoonacularAPIKey:  getEnv("SPOONACULAR_API_KEY", ""),
		GoogleAIBaseURL:    getEnv("GOOGLE_AI_BASE_URL", "https://generativelanguage.googleapis.com"),
		GoogleAIAPIKey:     getEnv("GOOGLE_AI_API_KEY", ""),
		GoogleAIModel:      getEnv("GOOGLE_AI_MODEL", "gemini-1.5-flash"),
	}

	if cfg.DBURL == "" {
		log.Println("DATABASE_URL is empty")
	}
	if cfg.JWTSecret == "" {
		log.Println("JWT_SECRET is empty")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}
