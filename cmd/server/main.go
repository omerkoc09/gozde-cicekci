package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/omerkoc/cicekci/internal/api/app"
	"github.com/omerkoc/cicekci/internal/api/idare"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
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
	catSvc := category.NewService(category.NewStore(pool))
	prodSvc := product.NewService(product.NewStore(pool))

	imgStore, err := newImageStore(cfg)
	if err != nil {
		log.Fatalf("görsel saklama: %v", err)
	}
	imgSvc := image.NewService(imgStore, image.NewDB(pool))

	isProduction := cfg.IsProduction()

	f := fiber.New(fiber.Config{
		AppName:               "cicekci",
		DisableStartupMessage: false,
		BodyLimit:             10 * 1024 * 1024, // Plan 2'de görsel yükleme için
	})

	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins(cfg.SiteURL, isProduction),
		AllowCredentials: true, // cookie gönderimi için zorunlu
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
	}))

	f.Get("/health", func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db down"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// apiGroup — "api" adı internal/api paketiyle çakışırdı.
	apiGroup := f.Group("/api")
	app.Register(apiGroup, catSvc, prodSvc, imgSvc)
	idare.Register(apiGroup.Group("/admin"), idare.Deps{
		AuthSvc:      authSvc,
		CatSvc:       catSvc,
		ProdSvc:      prodSvc,
		ImgSvc:       imgSvc,
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
