package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                     string
	DatabaseURL                string
	RedisURL                   string
	JWTSecret                  string
	GoogleClientID             string
	BaseURL                    string
	AllowedOrigins             []string
	TrustedProxies             []string
	Port                       string
	RunMigrations              bool
	MigrationsDir              string
	AuthRateLimitPerMinute     int
	APIRateLimitPerMinute      int
	RedirectRateLimitPerMinute int
	LinkCacheTTLSeconds        int
	RefreshCookieSecure        bool
	RefreshCookieDomain        string
}

func Load() *Config {
	appEnv := strings.ToLower(getEnv("APP_ENV", "development"))
	if appEnv == "" {
		appEnv = "development"
	}
	isProduction := appEnv == "production"

	cfg := &Config{
		AppEnv:                     appEnv,
		DatabaseURL:                getEnv("DATABASE_URL", "postgres://app:devpassword@localhost:5432/urlshortener?sslmode=disable"),
		RedisURL:                   getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:                  getEnv("JWT_SECRET", "dev-secret-change-in-prod"),
		GoogleClientID:             getEnv("GOOGLE_CLIENT_ID", ""),
		BaseURL:                    getEnv("BASE_URL", "http://localhost:8080"),
		AllowedOrigins:             splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173")),
		TrustedProxies:             splitCSV(getEnv("TRUSTED_PROXIES", "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")),
		Port:                       getEnv("PORT", "8080"),
		RunMigrations:              getEnv("RUN_MIGRATIONS", "false") == "true",
		MigrationsDir:              getEnv("MIGRATIONS_DIR", "/migrations"),
		AuthRateLimitPerMinute:     getEnvInt("AUTH_RATE_LIMIT_PER_MINUTE", 60),
		APIRateLimitPerMinute:      getEnvInt("API_RATE_LIMIT_PER_MINUTE", 300),
		RedirectRateLimitPerMinute: getEnvInt("REDIRECT_RATE_LIMIT_PER_MINUTE", 600),
		LinkCacheTTLSeconds:        getEnvInt("LINK_CACHE_TTL_SECONDS", 86400),
		RefreshCookieSecure:        getEnv("REFRESH_COOKIE_SECURE", boolDefault(isProduction)) == "true",
		RefreshCookieDomain:        getEnv("REFRESH_COOKIE_DOMAIN", ""),
	}

	if isProduction {
		mustBeSet("DATABASE_URL")
		mustBeSet("REDIS_URL")
		mustBeSet("JWT_SECRET")
		mustBeSet("GOOGLE_CLIENT_ID")
		mustBeSet("BASE_URL")
		mustBeSet("ALLOWED_ORIGINS")

		if strings.Contains(strings.ToLower(cfg.DatabaseURL), "sslmode=disable") {
			panic("DATABASE_URL must enable TLS in production (sslmode=disable is not allowed)")
		}

		if cfg.JWTSecret == "dev-secret-change-in-prod" || len(cfg.JWTSecret) < 32 {
			panic("JWT_SECRET must be a strong non-default secret in production (minimum 32 characters)")
		}
	}

	return cfg
}

func boolDefault(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func mustBeSet(key string) {
	if strings.TrimSpace(os.Getenv(key)) == "" {
		panic(fmt.Sprintf("%s must be set in production", key))
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
