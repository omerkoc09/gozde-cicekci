package idare

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// MaxUploadBytes tek görselin üst sınırı. Slider için 1920px+ yatay fotoğraf
// isteniyor (2400px'lik bir JPEG rahatlıkla 5-8MB olabilir), 10MB dardı.
//
// Fiber'in BodyLimit'i bunun ÜSTÜNDE olmalı — bkz. cmd/server/main.go.
// Eşit olursa multipart yükü (sınır satırları, alan adları) sınırı aşırır ve
// Fiber isteği handler'a hiç sokmadan 413 döner; o yanıtta CORS header'ı
// olmadığı için tarayıcı "dosya çok büyük" yerine anlamsız bir CORS hatası
// gösterir. Boşluk bırakılınca handler kendi anlaşılır mesajını dönebiliyor.
const MaxUploadBytes = 25 * 1024 * 1024 // 25MB

// maxUploadBytes paket içi kısa ad.
const maxUploadBytes = MaxUploadBytes

type imageHandler struct {
	svc     *image.Service
	prodSvc *product.Service
}

// badRequest 400 yanıtı üretir. Handler'larda tekrar eden kalıbı tek yere alır.
func badRequest(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
		Error: api.ErrorBody{Code: "invalid_input", Message: msg},
	})
}

// list GET /api/admin/products/:id/images
func (h *imageHandler) list(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	list, err := h.svc.ListByProduct(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toImageViews(h.svc, list))
}

// upload POST /api/admin/products/:id/images
// multipart/form-data, alan adı: "image"
func (h *imageHandler) upload(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	// Ürün var mı — yoksa yetim görsel yüklemeyelim.
	if _, err := h.prodSvc.GetByID(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return badRequest(c, "Görsel dosyası bulunamadı (alan adı: image)")
	}

	if fileHeader.Size > maxUploadBytes {
		return badRequest(c, fmt.Sprintf("Dosya çok büyük (en fazla %d MB)",
			maxUploadBytes/1024/1024))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return api.WriteError(c, fmt.Errorf("dosya açılamadı: %w", err))
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		return api.WriteError(c, fmt.Errorf("dosya okunamadı: %w", err))
	}

	img, err := h.svc.Upload(c.Context(), int64(id), data)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toImageView(h.svc, *img))
}

// delete DELETE /api/admin/images/:id
func (h *imageHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type reorderRequest struct {
	ImageIDs []int64 `json:"image_ids"`
}

// reorder PATCH /api/admin/products/:id/images/order
// Body: {"image_ids": [3, 1, 2]} — ürünün TÜM görsellerini içermeli.
// İlk id yeni kapak olur.
func (h *imageHandler) reorder(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req reorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.Reorder(c.Context(), int64(id), req.ImageIDs); err != nil {
		return api.WriteError(c, err)
	}

	list, err := h.svc.ListByProduct(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toImageViews(h.svc, list))
}
