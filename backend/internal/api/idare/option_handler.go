package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/productoption"
)

type optionHandler struct {
	svc *productoption.Service
}

func newOptionHandler(svc *productoption.Service) *optionHandler {
	return &optionHandler{svc: svc}
}

type createGroupRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// updateGroupRequest — kind YOK: tip oluşturulduktan sonra değişmez.
// sort_order da YOK: sıra reorder ucundan değişir.
type updateGroupRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type createValueRequest struct {
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
}

type updateValueRequest struct {
	Name      *string `json:"name"`
	SwatchHex *string `json:"swatch_hex"`
	IsActive  *bool   `json:"is_active"`
}

type optionReorderRequest struct {
	IDs []int64 `json:"ids"`
}

// list GET /api/admin/option-groups — pasifler dahil, değerleriyle.
func (h *optionHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListGroups(c.Context(), false)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionGroupViews(list))
}

// createGroup POST /api/admin/option-groups
func (h *optionHandler) createGroup(c *fiber.Ctx) error {
	var req createGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	g, err := h.svc.CreateGroup(c.Context(), productoption.CreateGroupInput{
		Name: req.Name,
		Kind: productoption.Kind(req.Kind),
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toOptionGroupView(*g))
}

// updateGroup PATCH /api/admin/option-groups/:id
func (h *optionHandler) updateGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	g, err := h.svc.UpdateGroup(c.Context(), int64(id), productoption.UpdateGroupInput{
		Name:     req.Name,
		IsActive: req.IsActive,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionGroupView(*g))
}

// deleteGroup DELETE /api/admin/option-groups/:id
func (h *optionHandler) deleteGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.DeleteGroup(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// productCount GET /api/admin/option-groups/:id/product-count
// Silme öncesi uyarı için: "Bu grup N üründe kullanılıyor".
func (h *optionHandler) productCount(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	n, err := h.svc.GroupProductCount(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(fiber.Map{"product_count": n})
}

// reorderGroups PUT /api/admin/option-groups/reorder
// Body: {"ids":[3,1,2]} — TÜM grupları içermeli.
func (h *optionHandler) reorderGroups(c *fiber.Ctx) error {
	var req optionReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.ReorderGroups(c.Context(), req.IDs); err != nil {
		return api.WriteError(c, err)
	}
	return h.list(c)
}

// createValue POST /api/admin/option-groups/:id/values
func (h *optionHandler) createValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req createValueRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	v, err := h.svc.CreateValue(c.Context(), productoption.CreateValueInput{
		GroupID:   int64(id),
		Name:      req.Name,
		SwatchHex: req.SwatchHex,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toOptionValueView(*v))
}

// reorderValues PUT /api/admin/option-groups/:id/values/reorder
func (h *optionHandler) reorderValues(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req optionReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.ReorderValues(c.Context(), int64(id), req.IDs); err != nil {
		return api.WriteError(c, err)
	}
	return h.list(c)
}

// updateValue PATCH /api/admin/option-values/:id
func (h *optionHandler) updateValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateValueRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	v, err := h.svc.UpdateValue(c.Context(), int64(id), productoption.UpdateValueInput{
		Name:      req.Name,
		SwatchHex: req.SwatchHex,
		IsActive:  req.IsActive,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionValueView(*v))
}

// deleteValue DELETE /api/admin/option-values/:id
func (h *optionHandler) deleteValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.DeleteValue(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
