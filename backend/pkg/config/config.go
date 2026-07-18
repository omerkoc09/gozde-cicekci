package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

	// Teslimat ayarları (spec §4). settings tablosu yerine config:
	// yılda bir değişir, ayarlar ekranı YAGNI (MVP §5.3 ile aynı gerekçe).
	// DEĞERLER ESNAFTAN ÖĞRENİLECEK — spec §7.
	DeliveryFee     string   // "50" — NUMERIC'e yazılacak, string tutuluyor (float precision)
	DeliverySlots   []string // ["09:00-12:00", ...]
	SameDayCutoff   string   // "16:00" — bu saatten sonra aynı güne sipariş yok
	MaxDeliveryDays int      // 30 — bugün + bu kadar güne kadar sipariş alınır

	// DeliveryDistricts teslimat yapılan ilçeler — bilgi amaçlı, ücrete
	// etkisi yok (2026-07-18 sipariş formu iyileştirmeleri spec'i).
	DeliveryDistricts []string

	// DeliveryFees ilçe adı → ücret eşlemesi ("Ödemiş" → "50"). Listede
	// olmayan ilçe genel DeliveryFee'ye düşer — ilçeye özel ücret esnaftan
	// öğrenildikçe eklenir, önceden hepsini girmek zorunluluğu yok.
	DeliveryFees map[string]string

	// PayTR ödeme (Faz 3). Anahtarlar boşsa mock provider kullanılır.
	PayTRMerchantID    string
	PayTRMerchantKey   string
	PayTRMerchantSalt  string
	PayTRTestMode      bool
	PaymentCallbackURL string
}

// envSearchDepth yukarı doğru kaç dizin taranacak. 4 fazlasıyla yeterli:
// backend/ → kök 1 seviye, `go test ./internal/...` en fazla 2-3 seviye içeride
// çalışır. Sınır olmasa kök dizine kadar tırmanıp alakasız bir .env bulabilirdi.
const envSearchDepth = 4

// findEnvFile çalışma dizininden başlayıp yukarı doğru .env arar.
//
// Gerekli çünkü .env repo kökünde ama Go kodu backend/ altında: `make run`
// backend/ içinde çalışıyor, `go test ./internal/slider` daha da derinde.
// godotenv yalnızca çalışma dizinine bakar, o yüzden dosyayı biz buluyoruz.
// .env kökte kalmalı — docker compose da onu okuyor.
func findEnvFile() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for i := 0; i < envSearchDepth; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir { // dosya sistemi kökü
			break
		}
		dir = parent
	}
	return "", false
}

// Load .env dosyasını okur (varsa) ve ortam değişkenlerinden Config üretir.
// .env yoksa hata değil — production'da değişkenler platformdan gelir.
func Load() (*Config, error) {
	// Hata yok sayılıyor: .env bulunamaması normal (prod).
	if path, ok := findEnvFile(); ok {
		_ = godotenv.Load(path)
	}

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

	loadDelivery(cfg)
	loadPayment(cfg)

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

// loadDelivery teslimat ayarlarını okur. Hepsinin makul varsayılanı var —
// esnaftan gerçek değerler öğrenilene kadar sistem çalışır (spec §7).
func loadDelivery(cfg *Config) {
	cfg.DeliveryFee = os.Getenv("DELIVERY_FEE")
	if cfg.DeliveryFee == "" {
		cfg.DeliveryFee = "0"
	}

	slots := os.Getenv("DELIVERY_SLOTS")
	if slots == "" {
		slots = "09:00-12:00,12:00-15:00,15:00-18:00"
	}
	cfg.DeliverySlots = strings.Split(slots, ",")
	for i := range cfg.DeliverySlots {
		cfg.DeliverySlots[i] = strings.TrimSpace(cfg.DeliverySlots[i])
	}

	cfg.SameDayCutoff = os.Getenv("SAME_DAY_CUTOFF")
	if cfg.SameDayCutoff == "" {
		cfg.SameDayCutoff = "16:00"
	}

	cfg.MaxDeliveryDays = 30
	if v := os.Getenv("MAX_DELIVERY_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxDeliveryDays = n
		}
	}

	districts := os.Getenv("DELIVERY_DISTRICTS")
	if districts == "" {
		districts = "Ödemiş,Tire,Bayındır,Kiraz,Beydağ"
	}
	cfg.DeliveryDistricts = strings.Split(districts, ",")
	for i := range cfg.DeliveryDistricts {
		cfg.DeliveryDistricts[i] = strings.TrimSpace(cfg.DeliveryDistricts[i])
	}

	cfg.DeliveryFees = map[string]string{}
	for _, pair := range strings.Split(os.Getenv("DELIVERY_FEES"), ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		ilce, ucret, ok := strings.Cut(pair, ":")
		if !ok {
			continue // bozuk giriş — sessizce atla, genel ücrete düşülür
		}

		cfg.DeliveryFees[strings.TrimSpace(ilce)] = strings.TrimSpace(ucret)
	}
}

// loadPayment PayTR ödeme ayarlarını okur. Anahtarlar boşsa PayTRConfigured
// false döner ve main.go mock provider'a düşer — geliştirmede gerçek PayTR
// hesabı olmadan da sistem çalışır.
func loadPayment(cfg *Config) {
	cfg.PayTRMerchantID = os.Getenv("PAYTR_MERCHANT_ID")
	cfg.PayTRMerchantKey = os.Getenv("PAYTR_MERCHANT_KEY")
	cfg.PayTRMerchantSalt = os.Getenv("PAYTR_MERCHANT_SALT")
	cfg.PayTRTestMode = os.Getenv("PAYTR_TEST_MODE") != "0" // varsayılan test modu açık
	cfg.PaymentCallbackURL = os.Getenv("PAYMENT_CALLBACK_URL")
	if cfg.PaymentCallbackURL == "" {
		cfg.PaymentCallbackURL = cfg.SiteURL + "/api/payment/callback"
	}
}

// PayTRConfigured gerçek PayTR anahtarları var mı. false ise main.go mock
// provider kullanır — geliştirme/test ortamında gerçek PayTR hesabı gerekmez.
func (c *Config) PayTRConfigured() bool {
	return c.PayTRMerchantID != "" && c.PayTRMerchantKey != "" && c.PayTRMerchantSalt != ""
}
