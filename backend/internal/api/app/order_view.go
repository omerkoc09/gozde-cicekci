package app

import "github.com/omerkoc/cicekci/internal/order"

// createOrderRequest — FİYAT ALANI YOK. Sunucu fiyatı DB'den okur (spec §2.2).
type createOrderRequest struct {
	Items []struct {
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
	} `json:"items"`

	Buyer struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	} `json:"buyer"`

	Recipient struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"recipient"`

	Delivery struct {
		Address string `json:"address"`
		Date    string `json:"date"` // "2026-07-20"
		Slot    string `json:"slot"`
	} `json:"delivery"`

	CardMessage string `json:"card_message"`
}

// createOrderResponse public yanıt — sipariş no ve toplam yeter.
// Müşteriye iç detay (id, statü) sızdırılmaz.
type createOrderResponse struct {
	OrderNo string `json:"order_no"`
	Total   string `json:"total"`
}

func toCreateOrderResponse(o *order.Order) createOrderResponse {
	return createOrderResponse{
		OrderNo: o.OrderNo,
		Total:   o.Total.StringFixed(2),
	}
}

// deliveryConfigResponse frontend'in saat/ücret hardcode etmemesi için.
// Sunucu ve frontend AYNI kaynaktan beslenmeli (spec §4).
type deliveryConfigResponse struct {
	Fee           string   `json:"fee"`
	Slots         []string `json:"slots"`
	SameDayCutoff string   `json:"same_day_cutoff"`
	MaxDays       int      `json:"max_days"`
}
