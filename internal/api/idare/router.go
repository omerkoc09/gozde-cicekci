package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

type Deps struct {
	AuthSvc      *auth.Service
	CatSvc       *category.Service
	ProdSvc      *product.Service
	ImgSvc       *image.Service
	JWTSecret    string
	SecureCookie bool
}

// Register admin rotalarını bağlar. /login hariç hepsi JWT korumalı.
func Register(router fiber.Router, d Deps) {
	ah := &authHandler{svc: d.AuthSvc, secureCookie: d.SecureCookie}
	ch := &categoryHandler{svc: d.CatSvc}
	ph := &productHandler{svc: d.ProdSvc, imgSvc: d.ImgSvc}
	ih := &imageHandler{svc: d.ImgSvc, prodSvc: d.ProdSvc}

	router.Post("/login", ah.login)

	protected := router.Group("", auth.Middleware(d.JWTSecret))

	protected.Post("/logout", ah.logout)
	protected.Get("/me", ah.me)

	protected.Get("/products", ph.list)
	protected.Post("/products", ph.create)
	protected.Get("/products/:id", ph.get)
	protected.Patch("/products/:id", ph.update)
	protected.Delete("/products/:id", ph.delete)

	protected.Get("/products/:id/images", ih.list)
	protected.Post("/products/:id/images", ih.upload)
	protected.Patch("/products/:id/images/order", ih.reorder)
	protected.Delete("/images/:id", ih.delete)

	protected.Get("/categories", ch.list)
	protected.Post("/categories", ch.create)
	protected.Patch("/categories/:id", ch.update)
	protected.Get("/categories/:id/product-count", ch.productCount)
	protected.Delete("/categories/:id", ch.delete)
}
