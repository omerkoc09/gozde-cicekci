package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/slider"
)

// Register public rotaları bağlar. Auth yok — herkes erişebilir.
func Register(router fiber.Router, catSvc *category.Service,
	prodSvc *product.Service, imgSvc *image.Service, sliderSvc *slider.Service,
	orderSvc *order.Service, deliveryCfg order.DeliveryConfig) {
	ch := &categoryHandler{svc: catSvc, imgSvc: imgSvc}
	ph := &productHandler{svc: prodSvc, imgSvc: imgSvc}
	sh := &sliderHandler{svc: sliderSvc, imgSvc: imgSvc}
	oh := &orderHandler{svc: orderSvc, cfg: deliveryCfg}

	router.Get("/slides", sh.list)

	router.Get("/products", ph.list)
	router.Get("/products/:slug", ph.getBySlug)

	// /categories/featured, /categories/:slug'dan ÖNCE tanımlanmalı —
	// yoksa "featured" slug olarak yakalanır.
	router.Get("/categories/featured", ch.listFeatured)
	router.Get("/categories", ch.list)
	router.Get("/categories/:slug", ch.getBySlug)

	router.Post("/orders", oh.create)
	router.Get("/delivery-config", oh.deliveryConfig)
}
