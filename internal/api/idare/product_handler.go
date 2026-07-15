package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/shopspring/decimal"
)

type productHandler struct {
	svc *product.Service
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	IsActive    *bool   `json:"is_active"`
	CategoryIDs []int64 `json:"category_ids"`
}

type updateProductRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *string `json:"price"`
	IsActive    *bool   `json:"is_active"`
	CategoryIDs []int64 `json:"category_ids"`
}

// list GET /api/admin/products — pasifler dahil
func (h *productHandler) list(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 24)
	offset := 0
	if page := c.QueryInt("page", 1); page > 1 {
		offset = (page - 1) * limit
	}

	list, err := h.svc.ListAdmin(c.Context(), limit, offset)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductViews(list))
}

// get GET /api/admin/products/:id — pasif olsa da döner
func (h *productHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	p, err := h.svc.GetByID(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductView(*p))
}

// create POST /api/admin/products
func (h *productHandler) create(c *fiber.Ctx) error {
	var req createProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz fiyat"},
		})
	}

	in := product.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		IsActive:    true, // varsayılan aktif (spec §3.2)
		CategoryIDs: req.CategoryIDs,
	}
	if req.IsActive != nil {
		in.IsActive = *req.IsActive
	}

	p, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toProductView(*p))
}

// update PATCH /api/admin/products/:id
func (h *productHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	var req updateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	in := product.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		CategoryIDs: req.CategoryIDs,
	}

	if req.Price != nil {
		price, err := decimal.NewFromString(*req.Price)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
				Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz fiyat"},
			})
		}
		in.Price = &price
	}

	p, err := h.svc.Update(c.Context(), int64(id), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductView(*p))
}

// delete DELETE /api/admin/products/:id
func (h *productHandler) delete(c *fiber.Ctx) error {
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
