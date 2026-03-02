package config

import "testing"

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg := Load()
	if cfg.AppEnv != "development" {
		t.Fatalf("expected development app env, got %q", cfg.AppEnv)
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.JWTSecret == "" {
		t.Fatalf("expected development fallbacks to be populated")
	}
}

func TestLoadProductionRequiresEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for missing production env")
		}
	}()

	_ = Load()
}

func TestLoadProductionRejectsDisabledTLS(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://app:secret@db:5432/urlshortener?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://redis:6379/0")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("BASE_URL", "https://flowlinks.example")
	t.Setenv("ALLOWED_ORIGINS", "https://flowlinks.example")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when DATABASE_URL disables TLS in production")
		}
	}()

	_ = Load()
}
