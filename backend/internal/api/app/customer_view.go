package app

import (
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateProfileRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	// Şifre değiştirme opsiyonel — ikisi de doluysa şifre de değişir.
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type customerView struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func toCustomerView(c *customer.Customer) customerView {
	return customerView{ID: c.ID, Email: c.Email, Name: c.Name, Phone: c.Phone}
}

// createOrderCustomerOrderItemView müşteri sipariş geçmişi kalemi — public
// sadeleştirilmiş görünüm. idare/order_view.go'daki toOrderView'ın fiyat
// formatlama desenini (StringFixed(2)) izler.
type createOrderCustomerOrderItemView struct {
	ProductName string                `json:"product_name"`
	Quantity    int                   `json:"quantity"`
	Options     []OrderItemOptionView `json:"options"`
}

// createOrderCustomerView müşterinin kendi sipariş geçmişi görünümü.
// Buyer/recipient iç detayları YOK — müşteri zaten kendi bilgilerini bilir.
type createOrderCustomerView struct {
	OrderNo      string                             `json:"order_no"`
	Status       string                             `json:"status"`
	Total        string                             `json:"total"`
	DeliveryDate string                             `json:"delivery_date"`
	Items        []createOrderCustomerOrderItemView `json:"items"`
}

func toCreateOrderCustomerView(o *order.Order) createOrderCustomerView {
	items := make([]createOrderCustomerOrderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, createOrderCustomerOrderItemView{
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Options:     toOrderItemOptionViews(it.Options),
		})
	}
	return createOrderCustomerView{
		OrderNo:      o.OrderNo,
		Status:       string(o.Status),
		Total:        o.Total.StringFixed(2),
		DeliveryDate: o.DeliveryDate.Format("2006-01-02"),
		Items:        items,
	}
}

// toCreateOrderCustomerViews boş liste için nil DEĞİL [] döner (JSON null
// yerine boş dizi).
func toCreateOrderCustomerViews(list []order.Order) []createOrderCustomerView {
	out := make([]createOrderCustomerView, 0, len(list))
	for i := range list {
		out = append(out, toCreateOrderCustomerView(&list[i]))
	}
	return out
}
