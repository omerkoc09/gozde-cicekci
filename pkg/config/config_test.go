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

// setBaseEnv zorunlu değişkenleri set eder — saklama testleri için ortak kurulum.
func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "http://localhost:3000")
	t.Setenv("APP_ENV", "development")
}

func TestLoad_DefaultsToLocalStorage(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STORAGE_DRIVER", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "local", cfg.StorageDriver)
	assert.Equal(t, "./uploads", cfg.UploadDir)
	assert.Contains(t, cfg.UploadBaseURL, "/uploads")
}

// Eksik R2 ayarı sessizce geçilmemeli — production'da görsel yüklenemediğini
// fark etmek geç olur.
func TestLoad_R2ListsAllMissingSettings(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ACCOUNT_ID", "abc")
	t.Setenv("R2_ACCESS_KEY_ID", "")
	t.Setenv("R2_SECRET_ACCESS_KEY", "")
	t.Setenv("R2_BUCKET", "cicekci")
	t.Setenv("R2_PUBLIC_URL", "")

	_, err := Load()

	require.Error(t, err)
	// Hepsi tek seferde listelenmeli — tek tek keşfetmek zaman kaybı.
	assert.Contains(t, err.Error(), "R2_ACCESS_KEY_ID")
	assert.Contains(t, err.Error(), "R2_SECRET_ACCESS_KEY")
	assert.Contains(t, err.Error(), "R2_PUBLIC_URL")
	assert.NotContains(t, err.Error(), "R2_ACCOUNT_ID", "dolu olan listelenmemeli")
	assert.NotContains(t, err.Error(), "R2_BUCKET")
}

func TestLoad_R2WithAllSettings(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ACCOUNT_ID", "abc")
	t.Setenv("R2_ACCESS_KEY_ID", "key")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_BUCKET", "cicekci")
	t.Setenv("R2_PUBLIC_URL", "https://cdn.example.com")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "r2", cfg.StorageDriver)
	assert.Equal(t, "cicekci", cfg.R2Bucket)
}

func TestLoad_RejectsUnknownStorageDriver(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STORAGE_DRIVER", "gcs")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_DRIVER")
}
