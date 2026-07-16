package api

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// WriteError domain hatasını HTTP yanıtına çevirir.
// Bilinmeyen hatalar 500 olur ve detayı sızmaz — sadece log'a düşer.
func WriteError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errorsx.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: ErrorBody{Code: "not_found", Message: "Kayıt bulunamadı"},
		})
	case errors.Is(err, errorsx.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: ErrorBody{Code: "invalid_input", Message: err.Error()},
		})
	case errors.Is(err, errorsx.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error: ErrorBody{Code: "unauthorized", Message: "Yetkisiz"},
		})
	case errors.Is(err, errorsx.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error: ErrorBody{Code: "conflict", Message: "Bu kayıt zaten var"},
		})
	default:
		log.Printf("beklenmeyen hata: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: ErrorBody{Code: "internal", Message: "Sunucu hatası"},
		})
	}
}
