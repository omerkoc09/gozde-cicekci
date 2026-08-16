package idare

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

// sort_order alanı YOK: yeni kategori kendi ekseninin sonuna eklenir ve sıra
// yalnızca reorder ucundan değişir. Tek kategorinin sırasını elle yazmak
// listeyi tutarsız bırakırdı.
type createCategoryRequest struct {
	Name       string `json:"name"`
	Axis       string `json:"axis"`
	IsActive   *bool  `json:"is_active"`
	IsFeatured *bool  `json:"is_featured"`
}

type updateCategoryRequest struct {
	Name       *string `json:"name"`
	IsActive   *bool   `json:"is_active"`
	IsFeatured *bool   `json:"is_featured"`
}

// categoryReorderRequest bir eksenin yeni sırası.
type categoryReorderRequest struct {
	Axis string  `json:"axis"`
	IDs  []int64 `json:"ids"`
}

// list GET /api/admin/categories — pasifler dahil
func (h *categoryHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListAdmin(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(h.imgSvc, list))
}

// create POST /api/admin/categories
func (h *categoryHandler) create(c *fiber.Ctx) error {
	var req createCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
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

	cat, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toCategoryView(h.imgSvc, *cat))
}

// reorder PUT /api/admin/categories/reorder
// Body: {"axis": "occasion", "ids": [3, 1, 2]} — o EKSENİN tüm kategorilerini
// içermeli. İki eksen bağımsız sıralanır.
func (h *categoryHandler) reorder(c *fiber.Ctx) error {
	var req categoryReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.Reorder(c.Context(), category.Axis(req.Axis), req.IDs); err != nil {
		return api.WriteError(c, err)
	}

	list, err := h.svc.ListAdmin(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(h.imgSvc, list))
}

// update PATCH /api/admin/categories/:id
func (h *categoryHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	cat, err := h.svc.Update(c.Context(), int64(id), category.UpdateInput{
		Name:       req.Name,
		IsActive:   req.IsActive,
		IsFeatured: req.IsFeatured,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(h.imgSvc, *cat))
}

// replaceImage PUT /api/admin/categories/:id/image
// multipart/form-data, alan adı: "image"
func (h *categoryHandler) replaceImage(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return badRequest(c, "Görsel dosyası bulunamadı (alan adı: image)")
	}

	data, err := readUpload(fileHeader)
	if err != nil {
		return api.WriteError(c, err)
	}

	cat, err := h.svc.ReplaceImage(c.Context(), int64(id), data)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(h.imgSvc, *cat))
}

// deleteImage DELETE /api/admin/categories/:id/image
// Kategori silinmez, yalnızca görseli kalkar — kart yedek görsele döner.
func (h *categoryHandler) deleteImage(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	cat, err := h.svc.DeleteImage(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(h.imgSvc, *cat))
}

// productCount GET /api/admin/categories/:id/product-count
// Silme öncesi uyarı için: "Bu kategoride N ürün var" (spec §4.1).
func (h *categoryHandler) productCount(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
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
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
