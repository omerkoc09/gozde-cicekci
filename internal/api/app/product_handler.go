package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/product"
)

type productHandler struct {
	svc *product.Service
}

// list GET /api/products?amac=&tip=&page=
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

	list, err := h.svc.ListPublic(c.Context(), f)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductViews(list))
}

// getBySlug GET /api/products/:slug
// Slug eskiyse 301 ile güncel URL'e yönlendirir (spec §4.2).
func (h *productHandler) getBySlug(c *fiber.Ctx) error {
	p, redirectTo, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}

	if redirectTo != "" {
		return c.Redirect("/api/products/"+redirectTo, fiber.StatusMovedPermanently)
	}

	return c.JSON(toProductView(*p))
}
