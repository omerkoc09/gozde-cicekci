package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/category"
)

type categoryHandler struct {
	svc *category.Service
}

// list GET /api/categories?axis=occasion|type
func (h *categoryHandler) list(c *fiber.Ctx) error {
	var axis *category.Axis
	if raw := c.Query("axis"); raw != "" {
		a := category.Axis(raw)
		axis = &a
	}

	list, err := h.svc.ListPublic(c.Context(), axis)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// listFeatured GET /api/categories/featured
func (h *categoryHandler) listFeatured(c *fiber.Ctx) error {
	list, err := h.svc.ListFeatured(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// getBySlug GET /api/categories/:slug
func (h *categoryHandler) getBySlug(c *fiber.Ctx) error {
	cat, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(*cat))
}
