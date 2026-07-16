package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

type productHandler struct {
	svc    *product.Service
	imgSvc *image.Service
}

// list GET /api/products?amac=&tip=&one_cikan=&page=
func (h *productHandler) list(c *fiber.Ctx) error {
	f := product.Filter{
		Limit:  c.QueryInt("limit", 24),
		Offset: 0,
	}

	if page := c.QueryInt("page", 1); page > 1 {
		f.Offset = (page - 1) * f.Limit
	}
	if amac := c.Query("amac"); amac != "" {
		f.OccasionSlug = &amac
	}
	if tip := c.Query("tip"); tip != "" {
		f.TypeSlug = &tip
	}
	// one_cikan=true → ana sayfa vitrini. Verilmezse tüm aktif ürünler döner;
	// katalog sayfası öne çıkmaya göre filtrelemiyor.
	f.FeaturedOnly = c.QueryBool("one_cikan", false)

	list, err := h.svc.ListPublic(c.Context(), f)
	if err != nil {
		return api.WriteError(c, err)
	}

	ids := make([]int64, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	grouped, err := h.imgSvc.ListByProducts(c.Context(), ids)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductViews(list, h.imgSvc, grouped))
}

// getBySlug GET /api/products/:slug
// Slug eskiyse 301 ile güncel URL'e yönlendirir (spec §4.2) — WhatsApp'ta
// paylaşılmış eski linkler ölmemeli.
func (h *productHandler) getBySlug(c *fiber.Ctx) error {
	p, redirectTo, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}

	if redirectTo != "" {
		return c.Redirect("/api/products/"+redirectTo, fiber.StatusMovedPermanently)
	}

	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductView(*p, h.imgSvc, imgs))
}
