package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// .env repo kökünde ama Go kodu backend/ altında çalışıyor — Load'ın yukarı
// doğru araması gerekiyor. Bu testler gerçek repo yerleşimine bağlı değil:
// geçici dizinde kendi ağaçlarını kuruyorlar.
func TestFindEnvFile_FindsInParentDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env"), []byte("X=1"), 0o644))

	sub := filepath.Join(root, "backend", "pkg", "config")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	t.Chdir(sub)

	path, ok := findEnvFile()

	require.True(t, ok, "üst dizindeki .env bulunmalı")
	// TempDir macOS'ta /var → /private/var symlink'i olabilir; sonu karşılaştır.
	assert.Equal(t, ".env", filepath.Base(path))
}

func TestFindEnvFile_PrefersClosest(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env"), []byte("X=kok"), 0o644))

	sub := filepath.Join(root, "backend")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, ".env"), []byte("X=yakin"), 0o644))
	t.Chdir(sub)

	path, ok := findEnvFile()

	require.True(t, ok)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "X=yakin", string(data), "en yakın .env kazanmalı")
}

// .env hiç yoksa hata değil — prod'da değişkenler platformdan geliyor.
func TestFindEnvFile_MissingIsNotFound(t *testing.T) {
	t.Chdir(t.TempDir())

	_, ok := findEnvFile()

	assert.False(t, ok)
}

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

func TestLoad_DeliveryDefaults(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "0", cfg.DeliveryFee)
	assert.Equal(t, []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"}, cfg.DeliverySlots)
	assert.Equal(t, "16:00", cfg.SameDayCutoff)
	assert.Equal(t, 30, cfg.MaxDeliveryDays)
	assert.Equal(t, []string{"Ödemiş", "Tire", "Bayındır", "Kiraz", "Beydağ"}, cfg.DeliveryDistricts)
	assert.Empty(t, cfg.DeliveryFees)
}

func TestLoad_DeliveryFromEnv(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DELIVERY_FEE", "50")
	t.Setenv("DELIVERY_SLOTS", "10:00-14:00,14:00-18:00")
	t.Setenv("SAME_DAY_CUTOFF", "15:30")
	t.Setenv("MAX_DELIVERY_DAYS", "14")
	t.Setenv("DELIVERY_DISTRICTS", "Ödemiş,Tire")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "50", cfg.DeliveryFee)
	assert.Equal(t, []string{"10:00-14:00", "14:00-18:00"}, cfg.DeliverySlots)
	assert.Equal(t, "15:30", cfg.SameDayCutoff)
	assert.Equal(t, 14, cfg.MaxDeliveryDays)
	assert.Equal(t, []string{"Ödemiş", "Tire"}, cfg.DeliveryDistricts)
}

func TestLoad_DeliveryFeesFromEnv(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DELIVERY_FEES", "Ödemiş:50,Tire:80,Bayındır:60")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"Ödemiş":   "50",
		"Tire":     "80",
		"Bayındır": "60",
	}, cfg.DeliveryFees)
}

func TestLoad_DeliveryFeesEmptyWhenUnset(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Empty(t, cfg.DeliveryFees)
}

// Bozuk giriş (iki nokta üst üste yok) sessizce atlanır — esnaf yanlış
// yazarsa tüm siparişleri kilitlemek yerine genel ücrete düşülür.
func TestLoad_DeliveryFeesIgnoresMalformedEntries(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DELIVERY_FEES", "Ödemiş:50,BozukGiriş,Tire:80")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"Ödemiş": "50",
		"Tire":   "80",
	}, cfg.DeliveryFees)
}

// PayTR anahtarları boşsa PayTRConfigured false döner — main.go bu durumda
// mock provider'a düşer, geliştirme ortamında gerçek PayTR hesabı gerekmez.
func TestLoad_PayTRDefaultsUnconfigured(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.False(t, cfg.PayTRConfigured())
	assert.True(t, cfg.PayTRTestMode, "PAYTR_TEST_MODE ayarlanmamışsa varsayılan test modu açık olmalı")
	assert.Equal(t, cfg.SiteURL+"/api/payment/callback", cfg.PaymentCallbackURL)
}

// Üçü de doluysa PayTRConfigured true döner — main.go gerçek PayTR provider'ı seçer.
func TestLoad_PayTRConfiguredWhenAllKeysSet(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("PAYTR_MERCHANT_ID", "123")
	t.Setenv("PAYTR_MERCHANT_KEY", "key")
	t.Setenv("PAYTR_MERCHANT_SALT", "salt")
	t.Setenv("PAYTR_TEST_MODE", "0")

	cfg, err := Load()
	require.NoError(t, err)

	assert.True(t, cfg.PayTRConfigured())
	assert.False(t, cfg.PayTRTestMode)
}

// Kısmi anahtar seti (biri eksik) hâlâ unconfigured sayılmalı — PayTR'a
// yarım kimlik bilgisiyle istek atmak anlamsız hata mesajlarına yol açar.
func TestLoad_PayTRUnconfiguredWhenPartiallySet(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("PAYTR_MERCHANT_ID", "123")
	t.Setenv("PAYTR_MERCHANT_KEY", "key")
	// PAYTR_MERCHANT_SALT eksik.

	cfg, err := Load()
	require.NoError(t, err)

	assert.False(t, cfg.PayTRConfigured())
}

// PAYMENT_CALLBACK_URL açıkça verilmişse SiteURL'den türetilen varsayılanı
// geçersiz kılmalı — PayTR panelindeki bildirim URL'i özelleştirilebilmeli.
func TestLoad_PaymentCallbackURLOverride(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("PAYMENT_CALLBACK_URL", "https://example.com/ozel/geri-bildirim")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/ozel/geri-bildirim", cfg.PaymentCallbackURL)
}
