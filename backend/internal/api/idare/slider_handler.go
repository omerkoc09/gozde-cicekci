package idare

import (
	"fmt"
	"io"
	"mime/multipart"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/slider"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type sliderHandler struct {
	svc    *slider.Service
	imgSvc *image.Service
}

// readUpload multipart dosyayı boyut sınırıyla okur.
func readUpload(fh *multipart.FileHeader) ([]byte, error) {
	if fh.Size > maxUploadBytes {
		return nil, fmt.Errorf("%w: dosya çok büyük (en fazla %d MB)",
			errorsx.ErrInvalidInput, maxUploadBytes/1024/1024)
	}

	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("dosya açılamadı: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		return nil, fmt.Errorf("dosya okunamadı: %w", err)
	}
	return data, nil
}

// list GET /api/admin/slides — pasifler dahil
func (h *sliderHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListAdmin(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toSlideViews(h.imgSvc, list))
}

// create POST /api/admin/slides
// multipart/form-data: title, subtitle, is_active, sort_order + image
// Görsel ve metin tek istekte gelir — görselsiz slayt oluşturulamaz.
func (h *sliderHandler) create(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return badRequest(c, "Görsel dosyası bulunamadı (alan adı: image)")
	}

	data, err := readUpload(fileHeader)
	if err != nil {
		return api.WriteError(c, err)
	}

	in := slider.CreateInput{
		Title:    c.FormValue("title"),
		Subtitle: c.FormValue("subtitle"),
		IsActive: true, // varsayılan aktif
	}

	if v := c.FormValue("is_active"); v != "" {
		in.IsActive = v == "true" || v == "1"
	}

	// sort_order verilmezse sona eklenir — panelde her yeni slayt için
	// sıra düşünmek zorunda kalmasın.
	if v := c.FormValue("sort_order"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return badRequest(c, "Geçersiz sıra")
		}
		in.SortOrder = n
	} else {
		next, err := h.svc.NextSortOrder(c.Context())
		if err != nil {
			return api.WriteError(c, err)
		}
		in.SortOrder = next
	}

	slide, err := h.svc.Create(c.Context(), in, data)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toSlideView(h.imgSvc, *slide))
}

type updateSlideRequest struct {
	Title     *string `json:"title"`
	Subtitle  *string `json:"subtitle"`
	IsActive  *bool   `json:"is_active"`
	SortOrder *int    `json:"sort_order"`
}

// update PATCH /api/admin/slides/:id — yalnızca metin alanları.
// Görsel değişimi ayrı uçta (replaceImage): eski dosyanın silinmesi gerekiyor.
func (h *sliderHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateSlideRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	slide, err := h.svc.Update(c.Context(), int64(id), slider.UpdateInput{
		Title:     req.Title,
		Subtitle:  req.Subtitle,
		IsActive:  req.IsActive,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toSlideView(h.imgSvc, *slide))
}

// replaceImage PUT /api/admin/slides/:id/image
// multipart/form-data, alan adı: "image"
func (h *sliderHandler) replaceImage(c *fiber.Ctx) error {
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

	slide, err := h.svc.ReplaceImage(c.Context(), int64(id), data)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toSlideView(h.imgSvc, *slide))
}

// delete DELETE /api/admin/slides/:id
func (h *sliderHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
