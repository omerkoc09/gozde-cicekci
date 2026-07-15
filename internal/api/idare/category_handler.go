package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/category"
)

type categoryHandler struct {
	svc *category.Service
}

type createCategoryRequest struct {
	Name       string `json:"name"`
	Axis       string `json:"axis"`
	IsActive   *bool  `json:"is_active"`
	IsFeatured *bool  `json:"is_featured"`
	SortOrder  *int   `json:"sort_order"`
}

type updateCategoryRequest struct {
	Name       *string `json:"name"`
	IsActive   *bool   `json:"is_active"`
	IsFeatured *bool   `json:"is_featured"`
	SortOrder  *int    `json:"sort_order"`
}

// list GET /api/admin/categories — pasifler dahil
func (h *categoryHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListAdmin(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// create POST /api/admin/categories
func (h *categoryHandler) create(c *fiber.Ctx) error {
	var req createCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	in := category.CreateInput{
		Name:     req.Name,
		Axis:     category.Axis(req.Axis),
		IsActive: true, // varsayılan aktif
	}
	if req.IsActive != nil {
		in.IsActive = *req.IsActive
	}
	if req.IsFeatured != nil {
		in.IsFeatured = *req.IsFeatured
	}
	if req.SortOrder != nil {
		in.SortOrder = *req.SortOrder
	}

	cat, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toCategoryView(*cat))
}

// update PATCH /api/admin/categories/:id
func (h *categoryHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	var req updateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	cat, err := h.svc.Update(c.Context(), int64(id), category.UpdateInput{
		Name:       req.Name,
		IsActive:   req.IsActive,
		IsFeatured: req.IsFeatured,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(*cat))
}

// productCount GET /api/admin/categories/:id/product-count
// Silme öncesi uyarı için: "Bu kategoride N ürün var" (spec §4.1).
func (h *categoryHandler) productCount(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	count, err := h.svc.ProductCount(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(fiber.Map{"product_count": count})
}

// delete DELETE /api/admin/categories/:id
// Junction kayıtları CASCADE ile gider, ürünler silinmez (spec §4.1).
func (h *categoryHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
