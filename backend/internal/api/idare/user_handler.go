package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/auth"
)

type userHandler struct {
	svc *auth.Service
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

type adminUserView struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// list GET /api/admin/users
func (h *userHandler) list(c *fiber.Ctx) error {
	users, err := h.svc.ListAdmins(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}

	views := make([]adminUserView, len(users))
	for i, u := range users {
		views[i] = adminUserView{ID: u.ID, Username: u.Username}
	}
	return c.JSON(views)
}

// create POST /api/admin/users
func (h *userHandler) create(c *fiber.Ctx) error {
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.CreateAdmin(c.Context(), req.Username, req.Password); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusCreated)
}

// delete DELETE /api/admin/users/:id
// Admin kendi hesabını silemez, son admin silinemez (spec: admin kullanıcı yönetimi §1).
func (h *userHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	requesterID, _ := c.Locals("userID").(int64)

	if err := h.svc.DeleteAdmin(c.Context(), requesterID, int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// resetPassword PATCH /api/admin/users/:id/password
func (h *userHandler) resetPassword(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.ChangePassword(c.Context(), int64(id), req.Password); err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
