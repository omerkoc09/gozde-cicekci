package order

import (
	"context"

	"github.com/omerkoc/cicekci/internal/payment"
)

// PaymentStarter order paketinin ödeme sağlayıcısından ihtiyaç duyduğu davranış.
// Somut implementasyon (PayTR/mock) main'de enjekte edilir.
type PaymentStarter interface {
	Start(ctx context.Context, in payment.StartInput) (payment.StartResult, error)
	VerifyCallback(in payment.CallbackInput) payment.CallbackResult
	Refund(ctx context.Context, in payment.RefundInput) error
}
