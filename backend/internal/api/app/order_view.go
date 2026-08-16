package app

import "github.com/omerkoc/cicekci/internal/order"

// createOrderRequest — FİYAT ALANI YOK. Sunucu fiyatı DB'den okur (spec §2.2).
type createOrderRequest struct {
	Items []struct {
		ProductID      int64   `json:"product_id"`
		Quantity       int     `json:"quantity"`
		OptionValueIDs []int64 `json:"option_value_ids"`
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
		Address  string `json:"address"`
		District string `json:"district"`
		Date     string `json:"date"` // "2026-07-20"
		Slot     string `json:"slot"`
	} `json:"delivery"`

	CardMessage string `json:"card_message"`
}

// createOrderResponse public yanıt — sipariş no, toplam ve PayTR iframe
// token'ı yeter. Müşteriye iç detay (id, statü) sızdırılmaz.
type createOrderResponse struct {
	OrderNo    string `json:"order_no"`
	Total      string `json:"total"`
	PaytrToken string `json:"paytr_token"`
}

func toCreateOrderResponse(o *order.Order, token string) createOrderResponse {
	return createOrderResponse{
		OrderNo:    o.OrderNo,
		Total:      o.Total.StringFixed(2),
		PaytrToken: token,
	}
}

// OrderItemOptionView sipariş kalemindeki seçim — sipariş anındaki
// kopyadan gelir, güncel gruba bakılmaz.
type OrderItemOptionView struct {
	GroupName string `json:"group_name"`
	ValueName string `json:"value_name"`
	SwatchHex string `json:"swatch_hex"`
}

func toOrderItemOptionViews(opts []order.OrderItemOption) []OrderItemOptionView {
	out := make([]OrderItemOptionView, 0, len(opts))
	for _, o := range opts {
		out = append(out, OrderItemOptionView{
			GroupName: o.GroupName,
			ValueName: o.ValueName,
			SwatchHex: o.SwatchHex,
		})
	}
	return out
}

// deliveryConfigResponse frontend'in saat/ücret hardcode etmemesi için.
// Sunucu ve frontend AYNI kaynaktan beslenmeli (spec §4).
type deliveryConfigResponse struct {
	Fee           string            `json:"fee"`
	Slots         []string          `json:"slots"`
	SameDayCutoff string            `json:"same_day_cutoff"`
	MaxDays       int               `json:"max_days"`
	Districts     []string          `json:"districts"`
	DistrictFees  map[string]string `json:"district_fees"`
}
