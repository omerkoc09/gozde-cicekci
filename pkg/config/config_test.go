package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("PORT", "9999")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "postgres://x/y", cfg.DatabaseURL)
	assert.Equal(t, "9999", cfg.Port)
	assert.Equal(t, "905551234567", cfg.WhatsAppNumber)
}

func TestLoad_DefaultsPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("PORT", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
}

func TestLoad_FailsWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_FailsWithShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "short")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_DefaultsToDevelopment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "http://localhost:3000")
	t.Setenv("APP_ENV", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.False(t, cfg.IsProduction())
}

func TestLoad_ProductionRequiresHTTPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "http://example.com")
	t.Setenv("APP_ENV", "production")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SITE_URL")
}

func TestLoad_ProductionWithHTTPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("APP_ENV", "production")

	cfg, err := Load()

	require.NoError(t, err)
	assert.True(t, cfg.IsProduction())
}

func TestLoad_RejectsUnknownAppEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("APP_ENV", "staging")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV")
}
