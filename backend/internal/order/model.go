package order

import (
	"time"

	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusAwaitingPayment Status = "awaiting_payment"
	StatusPaid            Status = "paid"
	StatusDelivered       Status = "delivered"
	StatusRefunded        Status = "refunded"
)

// Valid statü geçerli mi. DB'de CHECK var ama hatayı service'te yakalayıp
// düzgün mesaj vermek için burada da kontrol ediliyor.
func (s Status) Valid() bool {
	switch s {
	case StatusAwaitingPayment, StatusPaid, StatusDelivered, StatusRefunded:
		return true
	}
	return false
}

type OrderItem struct {
	ID           int64           `json:"id"`
	ProductID    *int64          `json:"product_id"` // ürün silinmişse nil
	ProductName  string          `json:"product_name"`
	PriceAtOrder decimal.Decimal `json:"price_at_order"`
	Quantity     int             `json:"quantity"`
}

type Order struct {
	ID      int64  `json:"id"`
	OrderNo string `json:"order_no"`
	Status  Status `json:"status"`

	BuyerName  string `json:"buyer_name"`
	BuyerPhone string `json:"buyer_phone"`
	BuyerEmail string `json:"buyer_email"`

	RecipientName    string    `json:"recipient_name"`
	RecipientPhone   string    `json:"recipient_phone"`
	DeliveryAddress  string    `json:"delivery_address"`
	DeliveryDistrict string    `json:"delivery_district"`
	DeliveryDate     time.Time `json:"delivery_date"`
	DeliverySlot     string    `json:"delivery_slot"`
	CardMessage      string    `json:"card_message"`

	ItemsTotal  decimal.Decimal `json:"items_total"`
	DeliveryFee decimal.Decimal `json:"delivery_fee"`
	Total       decimal.Decimal `json:"total"`

	PaidAt     *time.Time `json:"paid_at,omitempty"`
	RefundedAt *time.Time `json:"refunded_at,omitempty"`
	PaymentRef string     `json:"payment_ref,omitempty"`

	Note      string      `json:"note"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// NewOrder store'a giden kayıt — tutarlar service'te hesaplanmış,
// fiyatlar DB'den okunmuş halde gelir.
type NewOrder struct {
	BuyerName  string
	BuyerPhone string
	BuyerEmail string

	RecipientName    string
	RecipientPhone   string
	DeliveryAddress  string
	DeliveryDistrict string
	DeliveryDate     time.Time
	DeliverySlot     string
	CardMessage      string

	ItemsTotal  decimal.Decimal
	DeliveryFee decimal.Decimal
	Total       decimal.Decimal

	Items []NewOrderItem
}

type NewOrderItem struct {
	ProductID    int64
	ProductName  string
	PriceAtOrder decimal.Decimal
	Quantity     int
}
