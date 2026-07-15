package config

import (
	"fmt"
	"os"
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

	return cfg, nil
}

// IsProduction production ortamında mı çalıştığımızı söyler.
// Cookie'nin Secure bayrağı buna bağlı.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}
