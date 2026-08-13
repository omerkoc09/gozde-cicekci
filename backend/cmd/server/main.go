package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/api/app"
	"github.com/omerkoc/cicekci/internal/api/idare"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/slider"
	"github.com/omerkoc/cicekci/pkg/config"
	"github.com/omerkoc/cicekci/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanı: %v", err)
	}
	defer pool.Close()

	authSvc := auth.NewService(auth.NewStore(pool), cfg.JWTSecret)
	prodSvc := product.NewService(product.NewStore(pool))

	// imgSvc önce kurulmalı: kategori ve slider servisleri ona bağımlı.
	imgStore, err := newImageStore(cfg)
	if err != nil {
		log.Fatalf("görsel saklama: %v", err)
	}
	imgSvc := image.NewService(imgStore, image.NewDB(pool))

	catSvc := category.NewService(category.NewStore(pool), imgSvc)
	sliderSvc := slider.NewService(slider.NewStore(pool), imgSvc)

	deliveryCfg := order.DeliveryConfig{
		Fee:           cfg.DeliveryFee,
		Slots:         cfg.DeliverySlots,
		SameDayCutoff: cfg.SameDayCutoff,
		MaxDays:       cfg.MaxDeliveryDays,
		Districts:     cfg.DeliveryDistricts,
		DistrictFees:  cfg.DeliveryFees,
	}
	var payProvider order.PaymentStarter
	if cfg.PayTRConfigured() {
		payProvider = payment.NewPayTR(payment.PayTRConfig{
			MerchantID:   cfg.PayTRMerchantID,
			MerchantKey:  cfg.PayTRMerchantKey,
			MerchantSalt: cfg.PayTRMerchantSalt,
			TestMode:     cfg.PayTRTestMode,
		})
		log.Println("ödeme: PayTR provider aktif (test_mode:", cfg.PayTRTestMode, ")")
	} else {
		payProvider = payment.NewMockProvider()
		log.Println("ödeme: MOCK provider (PayTR anahtarları yok)")
	}

	// Public site ödeme sonrası buralara döner.
	okURL := cfg.SiteURL + "/siparis/tamam"
	failURL := cfg.SiteURL + "/siparis/hata"

	orderSvc := order.NewService(order.NewStore(pool), product.NewStore(pool),
		deliveryCfg, payProvider, okURL, failURL)

	custSvc := customer.NewService(customer.NewStore(pool), cfg.JWTSecret)

	isProduction := cfg.IsProduction()

	f := fiber.New(fiber.Config{
		AppName:               "cicekci",
		DisableStartupMessage: false,

		// Görsel sınırının ÜSTÜNDE: multipart yükü (sınır satırları, alan
		// adları) dosyanın üstüne birkaç yüz bayt ekliyor. Eşit olsalardı tam
		// sınırdaki dosya Fiber'e takılır, handler'a hiç ulaşmaz ve kullanıcı
		// "dosya çok büyük" yerine CORS hatası görürdü (413'te CORS header'ı
		// yok). Boşluk sayesinde reddi handler veriyor, mesaj anlaşılır oluyor.
		BodyLimit: idare.MaxUploadBytes + 2*1024*1024,

		ErrorHandler: errorHandler(allowedOrigins(cfg.SiteURL, cfg.IsProduction())),
	})

	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins(cfg.SiteURL, isProduction),
		AllowCredentials: true, // cookie gönderimi için zorunlu
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))

	f.Get("/health", func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db down"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// apiGroup — "api" adı internal/api paketiyle çakışırdı.
	apiGroup := f.Group("/api")
	app.Register(apiGroup, catSvc, prodSvc, imgSvc, sliderSvc, orderSvc, deliveryCfg,
		custSvc, cfg.JWTSecret, isProduction)
	idare.Register(apiGroup.Group("/admin"), idare.Deps{
		AuthSvc:      authSvc,
		CatSvc:       catSvc,
		ProdSvc:      prodSvc,
		ImgSvc:       imgSvc,
		SliderSvc:    sliderSvc,
		OrderSvc:     orderSvc,
		CustSvc:      custSvc,
		JWTSecret:    cfg.JWTSecret,
		SecureCookie: isProduction,
	})

	// Local saklama modunda görselleri statik servis et.
	// R2 modunda gerekmez — CDN servis ediyor.
	if cfg.StorageDriver == "local" {
		f.Static("/uploads", cfg.UploadDir, fiber.Static{
			MaxAge: 31536000, // key'ler rastgele, içerik değişmiyor
		})
	}

	go func() {
		if err := f.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("sunucu: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("kapatılıyor...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("kapatma hatası: %v", err)
	}
}

// errorHandler router'a hiç ulaşamayan hataları karşılar.
//
// Asıl derdi BodyLimit aşımı: Fiber isteği middleware zincirinden ÖNCE kesiyor,
// dolayısıyla cors middleware'i çalışmıyor ve 413 yanıtında Access-Control-*
// header'ı olmuyor. Tarayıcı da bunu "CORS policy" hatası diye gösteriyor —
// kullanıcı dosyasının büyük olduğunu asla öğrenemiyor. Header'ı burada elle
// ekleyip anlaşılır bir mesaj dönüyoruz.
func errorHandler(allowOrigins string) fiber.ErrorHandler {
	allowed := strings.Split(allowOrigins, ",")

	return func(c *fiber.Ctx, err error) error {
		// İstek origin'i izinliyse CORS header'ını geri koy — yoksa tarayıcı
		// yanıtı okuyamaz ve gerçek hata mesajı kaybolur.
		if origin := c.Get("Origin"); origin != "" {
			for _, a := range allowed {
				if strings.TrimSpace(a) == origin {
					c.Set("Access-Control-Allow-Origin", origin)
					c.Set("Access-Control-Allow-Credentials", "true")
					break
				}
			}
		}

		code := fiber.StatusInternalServerError
		var fe *fiber.Error
		if errors.As(err, &fe) {
			code = fe.Code
		}

		if code == fiber.StatusRequestEntityTooLarge {
			return c.Status(code).JSON(api.ErrorResponse{
				Error: api.ErrorBody{
					Code: "invalid_input",
					Message: fmt.Sprintf("Dosya çok büyük (en fazla %d MB)",
						idare.MaxUploadBytes/1024/1024),
				},
			})
		}

		log.Printf("beklenmeyen hata (%d): %v", code, err)
		return c.Status(code).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "internal", Message: "Sunucu hatası"},
		})
	}
}

// allowedOrigins CORS'a izin verilen origin listesini kurar.
//
// Prod'da yalnızca SITE_URL: admin paneli Caddy arkasında aynı origin'den
// (/api) servis edilir, cross-origin isteği yoktur.
//
// Development'ta admin paneli Vite dev sunucusunda (:5173) ayrı origin'de
// çalışır ve doğrudan :8080'e istek atar (frontend/idare/.env.development →
// VITE_API_BASE_URL=http://localhost:8080/api). Bu origin eklenmezse tarayıcı
// preflight'ı bloklar ve panele giriş yapılamaz.
func allowedOrigins(siteURL string, isProduction bool) string {
	if isProduction {
		return siteURL
	}
	return siteURL + ",http://localhost:5173"
}

// newImageStore config'e göre saklama implementasyonunu seçer.
// Uygulamanın geri kalanı hangisi olduğunu bilmez — spec §4.4'teki mimari
// kısıt burada somutlaşıyor: R2'den diske geçiş tek config satırı.
func newImageStore(cfg *config.Config) (image.Store, error) {
	switch cfg.StorageDriver {
	case "r2":
		return image.NewR2Store(image.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKey,
			SecretAccessKey: cfg.R2SecretKey,
			Bucket:          cfg.R2Bucket,
			PublicURL:       cfg.R2PublicURL,
		})
	case "local":
		return image.NewLocalStore(cfg.UploadDir, cfg.UploadBaseURL)
	default:
		return nil, fmt.Errorf("bilinmeyen STORAGE_DRIVER: %q", cfg.StorageDriver)
	}
}
