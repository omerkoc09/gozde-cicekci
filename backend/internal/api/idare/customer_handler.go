package idare

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
)

// customerHandler admin panelinde SALT OKUNUR müşteri görünümü. Oluşturma,
// düzenleme, silme, şifre sıfırlama YOK — kapsam kesin olarak buna kapalı.
type customerHandler struct {
	svc      *customer.Service
	orderSvc *order.Service
}

// customerView admin listesi/detayı — password_hash asla yok.
type customerView struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCustomerView(c *customer.Customer) customerView {
	return customerView{
		ID:        c.ID,
		Email:     c.Email,
		Name:      c.Name,
		Phone:     c.Phone,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

type customerListResponse struct {
	Items []customerView `json:"items"`
	Total int            `json:"total"`
}

// list GET /api/admin/customers?q=&page=&limit=
func (h *customerHandler) list(c *fiber.Ctx) error {
	q := c.Query("q")

	limit := c.QueryInt("limit", 50)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	list, err := h.svc.List(c.Context(), q, limit, offset)
	if err != nil {
		return api.WriteError(c, err)
	}

	total, err := h.svc.Count(c.Context(), q)
	if err != nil {
		return api.WriteError(c, err)
	}

	views := make([]customerView, len(list))
	for i, cst := range list {
		views[i] = toCustomerView(&cst)
	}

	return c.JSON(customerListResponse{Items: views, Total: total})
}

// customerDetailView profil + sipariş geçmişi (admin — salt okunur).
type customerDetailView struct {
	customerView
	Orders []orderView `json:"orders"`
}

// get GET /api/admin/customers/:id
func (h *customerHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	cst, err := h.svc.Get(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}

	orders, err := h.orderSvc.ListByCustomer(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(customerDetailView{
		customerView: toCustomerView(cst),
		Orders:       toOrderViews(orders),
	})
}
