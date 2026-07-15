package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	WhatsAppNumber string
	SiteURL        string
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

	return cfg, nil
}
