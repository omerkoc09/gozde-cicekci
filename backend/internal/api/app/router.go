package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/internal/slider"
)

// Register public rotaları bağlar. Auth yok — herkes erişebilir (müşteri
// uçları hariç, onlar customer.Middleware ile korunur).
func Register(router fiber.Router, catSvc *category.Service,
	prodSvc *product.Service, imgSvc *image.Service, sliderSvc *slider.Service,
	orderSvc *order.Service, deliveryCfg order.DeliveryConfig,
	custSvc *customer.Service, optSvc *productoption.Service,
	jwtSecret string, secureCookie bool) {
	ch := &categoryHandler{svc: catSvc, imgSvc: imgSvc}
	ph := &productHandler{svc: prodSvc, imgSvc: imgSvc, optSvc: optSvc}
	sh := &sliderHandler{svc: sliderSvc, imgSvc: imgSvc}
	oh := &orderHandler{svc: orderSvc, cfg: deliveryCfg, jwtSecret: jwtSecret}
	custH := &customerHandler{svc: custSvc, orderSvc: orderSvc, secureCookie: secureCookie}

	router.Get("/slides", sh.list)

	router.Get("/products", ph.list)
	router.Get("/products/:slug", ph.getBySlug)

	// /categories/featured, /categories/:slug'dan ÖNCE tanımlanmalı —
	// yoksa "featured" slug olarak yakalanır.
	router.Get("/categories/featured", ch.listFeatured)
	router.Get("/categories", ch.list)
	router.Get("/categories/:slug", ch.getBySlug)

	router.Post("/orders", oh.create)
	router.Post("/payment/callback", oh.paymentCallback)
	router.Get("/delivery-config", oh.deliveryConfig)

	router.Post("/customer/register", custH.register)
	router.Post("/customer/login", custH.login)
	router.Post("/customer/logout", custH.logout)

	// Auth korumalı müşteri uçları
	custProtected := router.Group("/customer", customer.Middleware(jwtSecret))
	custProtected.Get("/me", custH.me)
	custProtected.Patch("/me", custH.updateMe)
	custProtected.Get("/orders", custH.orders)
	custProtected.Get("/addresses", custH.addresses)
}
