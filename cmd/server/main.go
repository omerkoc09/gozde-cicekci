package main

import (
	"context"
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

	isProduction := cfg.IsProduction()

	f := fiber.New(fiber.Config{
		AppName:               "cicekci",
		DisableStartupMessage: false,
		BodyLimit:             10 * 1024 * 1024, // Plan 2'de görsel yükleme için
	})

	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.SiteURL,
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
	app.Register(apiGroup, catSvc, prodSvc)
	idare.Register(apiGroup.Group("/admin"), idare.Deps{
		AuthSvc:      authSvc,
		CatSvc:       catSvc,
		ProdSvc:      prodSvc,
		JWTSecret:    cfg.JWTSecret,
		SecureCookie: isProduction,
	})

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
