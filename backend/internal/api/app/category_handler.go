package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
)

type categoryHandler struct {
	svc    *category.Service
	imgSvc *image.Service
}

// axisQuery ?axis= parametresini okur. Yoksa nil — iki eksen de gelsin demek.
// Geçerlilik kontrolü service'te; burada sadece okuyoruz.
func axisQuery(c *fiber.Ctx) *category.Axis {
	raw := c.Query("axis")
	if raw == "" {
		return nil
	}
	a := category.Axis(raw)
	return &a
}

// list GET /api/categories?axis=occasion|type
func (h *categoryHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListPublic(c.Context(), axisQuery(c))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(h.imgSvc, list))
}

// listFeatured GET /api/categories/featured?axis=occasion|type
// Ana sayfa iki bölümü ayrı çekiyor: "Özel Günler" (occasion) ve
// "Çiçek Türlerine Göre" (type). axis verilmezse ikisi karışık döner.
func (h *categoryHandler) listFeatured(c *fiber.Ctx) error {
	list, err := h.svc.ListFeatured(c.Context(), axisQuery(c))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(h.imgSvc, list))
}

// getBySlug GET /api/categories/:slug
func (h *categoryHandler) getBySlug(c *fiber.Ctx) error {
	cat, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(h.imgSvc, *cat))
}
