package idare

import (
	"time"

	"github.com/omerkoc/cicekci/internal/order"
)

// orderItemView admin görünümü — esnaf ne göndereceğini görmeli.
type orderItemView struct {
	ProductID    *int64 `json:"product_id"`
	ProductName  string `json:"product_name"`
	PriceAtOrder string `json:"price_at_order"`
	Quantity     int    `json:"quantity"`
}

// orderView admin tam görünüm — public'ten farklı: her şey görünür.
type orderView struct {
	ID      int64  `json:"id"`
	OrderNo string `json:"order_no"`
	Status  string `json:"status"`

	BuyerName  string `json:"buyer_name"`
	BuyerPhone string `json:"buyer_phone"`
	BuyerEmail string `json:"buyer_email"`

	RecipientName    string `json:"recipient_name"`
	RecipientPhone   string `json:"recipient_phone"`
	DeliveryAddress  string `json:"delivery_address"`
	DeliveryDistrict string `json:"delivery_district"`
	DeliveryDate     string `json:"delivery_date"`
	DeliverySlot     string `json:"delivery_slot"`
	CardMessage      string `json:"card_message"`

	ItemsTotal  string `json:"items_total"`
	DeliveryFee string `json:"delivery_fee"`
	Total       string `json:"total"`

	PaidAt     *time.Time `json:"paid_at"`
	RefundedAt *time.Time `json:"refunded_at"`
	PaymentRef string     `json:"payment_ref"`

	Note      string          `json:"note"`
	Items     []orderItemView `json:"items"`
	CreatedAt time.Time       `json:"created_at"`
}

func toOrderView(o *order.Order) orderView {
	items := make([]orderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemView{
			ProductID:    it.ProductID,
			ProductName:  it.ProductName,
			PriceAtOrder: it.PriceAtOrder.StringFixed(2),
			Quantity:     it.Quantity,
		})
	}

	return orderView{
		ID:               o.ID,
		OrderNo:          o.OrderNo,
		Status:           string(o.Status),
		BuyerName:        o.BuyerName,
		BuyerPhone:       o.BuyerPhone,
		BuyerEmail:       o.BuyerEmail,
		RecipientName:    o.RecipientName,
		RecipientPhone:   o.RecipientPhone,
		DeliveryAddress:  o.DeliveryAddress,
		DeliveryDistrict: o.DeliveryDistrict,
		DeliveryDate:     o.DeliveryDate.Format("2006-01-02"),
		DeliverySlot:     o.DeliverySlot,
		CardMessage:      o.CardMessage,
		ItemsTotal:       o.ItemsTotal.StringFixed(2),
		DeliveryFee:      o.DeliveryFee.StringFixed(2),
		Total:            o.Total.StringFixed(2),
		PaidAt:           o.PaidAt,
		RefundedAt:       o.RefundedAt,
		PaymentRef:       o.PaymentRef,
		Note:             o.Note,
		Items:            items,
		CreatedAt:        o.CreatedAt,
	}
}

func toOrderViews(list []order.Order) []orderView {
	out := make([]orderView, 0, len(list))
	for i := range list {
		out = append(out, toOrderView(&list[i]))
	}
	return out
}

// updateOrderRequest PATCH: nil alan değişmez.
type updateOrderRequest struct {
	Status *string `json:"status"`
	Note   *string `json:"note"`
}
