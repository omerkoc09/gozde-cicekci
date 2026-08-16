package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/internal/slider"
)

type Deps struct {
	AuthSvc      *auth.Service
	CatSvc       *category.Service
	ProdSvc      *product.Service
	ImgSvc       *image.Service
	SliderSvc    *slider.Service
	OrderSvc     *order.Service
	CustSvc      *customer.Service
	OptSvc       *productoption.Service
	JWTSecret    string
	SecureCookie bool
}

// Register admin rotalarını bağlar. /login hariç hepsi JWT korumalı.
func Register(router fiber.Router, d Deps) {
	ah := &authHandler{svc: d.AuthSvc, secureCookie: d.SecureCookie}
	uh := &userHandler{svc: d.AuthSvc}
	ch := &categoryHandler{svc: d.CatSvc, imgSvc: d.ImgSvc}
	ph := &productHandler{svc: d.ProdSvc, imgSvc: d.ImgSvc}
	ih := &imageHandler{svc: d.ImgSvc, prodSvc: d.ProdSvc}
	sh := &sliderHandler{svc: d.SliderSvc, imgSvc: d.ImgSvc}
	oh := &orderHandler{svc: d.OrderSvc}
	cuh := &customerHandler{svc: d.CustSvc, orderSvc: d.OrderSvc}
	oph := newOptionHandler(d.OptSvc)

	router.Post("/login", ah.login)

	protected := router.Group("", auth.Middleware(d.JWTSecret))

	protected.Post("/logout", ah.logout)
	protected.Get("/me", ah.me)

	protected.Get("/users", uh.list)
	protected.Post("/users", uh.create)
	protected.Delete("/users/:id", uh.delete)
	protected.Patch("/users/:id/password", uh.resetPassword)

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
	// reorder ":id" kalıplarından ÖNCE — Fiber sıralı eşleştirir, sonra
	// gelseydi "/categories/reorder" isteği ":id" route'una düşer ve
	// "reorder" geçersiz id olarak reddedilirdi.
	protected.Put("/categories/reorder", ch.reorder)
	protected.Patch("/categories/:id", ch.update)
	protected.Get("/categories/:id/product-count", ch.productCount)
	protected.Put("/categories/:id/image", ch.replaceImage)
	protected.Delete("/categories/:id/image", ch.deleteImage)
	protected.Delete("/categories/:id", ch.delete)

	protected.Get("/slides", sh.list)
	protected.Post("/slides", sh.create)
	protected.Put("/slides/reorder", sh.reorder) // ":id" kalıplarından önce
	protected.Patch("/slides/:id", sh.update)
	protected.Put("/slides/:id/image", sh.replaceImage)
	protected.Delete("/slides/:id", sh.delete)

	protected.Get("/orders", oh.list)
	protected.Get("/orders/:id", oh.get)
	protected.Patch("/orders/:id", oh.update)
	protected.Post("/orders/:id/refund", oh.refund)

	// Müşteriler — salt okunur (spec: admin müşteri yönetimi). Oluşturma,
	// düzenleme, silme YOK; kapsam kesinlikle bununla sınırlı.
	protected.Get("/customers", cuh.list)
	protected.Get("/customers/:id", cuh.get)

	// reorder ":id" kalıplarından ÖNCE — Fiber sıralı eşleştirir.
	protected.Put("/option-groups/reorder", oph.reorderGroups)
	protected.Get("/option-groups", oph.list)
	protected.Post("/option-groups", oph.createGroup)
	protected.Patch("/option-groups/:id", oph.updateGroup)
	protected.Delete("/option-groups/:id", oph.deleteGroup)
	protected.Get("/option-groups/:id/product-count", oph.productCount)
	protected.Post("/option-groups/:id/values", oph.createValue)
	protected.Put("/option-groups/:id/values/reorder", oph.reorderValues)
	protected.Patch("/option-values/:id", oph.updateValue)
	protected.Delete("/option-values/:id", oph.deleteValue)
}
