package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	WhatsAppNumber string
	SiteURL        string
	AppEnv         string // "development" | "production"

	// Görsel saklama. StorageDriver "local" veya "r2".
	StorageDriver string
	UploadDir     string // local için
	UploadBaseURL string // local için
	R2AccountID   string
	R2AccessKey   string
	R2SecretKey   string
	R2Bucket      string
	R2PublicURL   string
}

// Load .env dosyasını okur (varsa) ve ortam değişkenlerinden Config üretir.
// .env yoksa hata değil — production'da değişkenler platformdan gelir.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		Port:           os.Getenv("PORT"),
		WhatsAppNumber: os.Getenv("WHATSAPP_NUMBER"),
		SiteURL:        os.Getenv("SITE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL zorunlu")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET en az 32 karakter olmalı")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	cfg.AppEnv = os.Getenv("APP_ENV")
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" {
		return nil, fmt.Errorf("geçersiz APP_ENV: %q (development veya production)", cfg.AppEnv)
	}
	// Production'da cookie Secure bayrağı açılır; bu yüzden site HTTPS olmalı.
	// Yanlış yapılandırmayı sessizce geçmek yerine burada yakalıyoruz.
	if cfg.AppEnv == "production" && !strings.HasPrefix(cfg.SiteURL, "https://") {
		return nil, fmt.Errorf("APP_ENV=production ise SITE_URL https:// ile başlamalı (şu an: %q)", cfg.SiteURL)
	}

	if err := loadStorage(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadStorage görsel saklama ayarlarını okur ve doğrular.
// Eksik R2 ayarını sessizce geçmek yerine burada yakalıyoruz — production'da
// görsel yüklenemediğini fark etmek geç olur.
func loadStorage(cfg *Config) error {
	cfg.StorageDriver = os.Getenv("STORAGE_DRIVER")
	if cfg.StorageDriver == "" {
		cfg.StorageDriver = "local"
	}
	cfg.UploadDir = os.Getenv("UPLOAD_DIR")
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}
	cfg.UploadBaseURL = os.Getenv("UPLOAD_BASE_URL")
	if cfg.UploadBaseURL == "" {
		cfg.UploadBaseURL = "http://localhost:" + cfg.Port + "/uploads"
	}
	cfg.R2AccountID = os.Getenv("R2_ACCOUNT_ID")
	cfg.R2AccessKey = os.Getenv("R2_ACCESS_KEY_ID")
	cfg.R2SecretKey = os.Getenv("R2_SECRET_ACCESS_KEY")
	cfg.R2Bucket = os.Getenv("R2_BUCKET")
	cfg.R2PublicURL = os.Getenv("R2_PUBLIC_URL")

	switch cfg.StorageDriver {
	case "local":
		return nil
	case "r2":
		missing := make([]string, 0, 5)
		for name, val := range map[string]string{
			"R2_ACCOUNT_ID":        cfg.R2AccountID,
			"R2_ACCESS_KEY_ID":     cfg.R2AccessKey,
			"R2_SECRET_ACCESS_KEY": cfg.R2SecretKey,
			"R2_BUCKET":            cfg.R2Bucket,
			"R2_PUBLIC_URL":        cfg.R2PublicURL,
		} {
			if val == "" {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return fmt.Errorf("STORAGE_DRIVER=r2 için eksik ayarlar: %s",
				strings.Join(missing, ", "))
		}
		return nil
	default:
		return fmt.Errorf("geçersiz STORAGE_DRIVER: %q (local veya r2)", cfg.StorageDriver)
	}
}

// IsProduction production ortamında mı çalıştığımızı söyler.
// Cookie'nin Secure bayrağı buna bağlı.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}
