package payment

import "context"

// BasketItem PayTR user_basket için tek satır.
type BasketItem struct {
	Name       string
	PriceKurus int64
	Quantity   int
}

// StartInput ödeme başlatmak için gereken her şey (sağlayıcıdan bağımsız).
type StartInput struct {
	MerchantOID string
	UserIP      string
	Email       string
	AmountKurus int64
	Basket      []BasketItem
	OkURL       string
	FailURL     string

	// PayTR canlı sunucusu user_name/user_address/user_phone alanlarını
	// zorunlu tutuyor (doküman "opsiyonel" dese de get-token bunlarsız
	// "Zorunlu alan degeri gecersiz veya gonderilmedi" hatası veriyor).
	UserName    string
	UserAddress string
	UserPhone   string
}

type StartResult struct {
	Token string
}

// CallbackInput sağlayıcının bildirim POST'undan gelen ham alanlar.
type CallbackInput struct {
	MerchantOID string
	Status      string
	TotalAmount string
	Hash        string
}

// CallbackResult VerifyCallback sonucu. OK yalnızca hash geçerli VE status=success.
type CallbackResult struct {
	OK          bool
	MerchantOID string
}

type RefundInput struct {
	MerchantOID       string
	ReturnAmountKurus int64
}

// Provider ödeme sağlayıcısı arayüzü. PayTR bunu paytr.go'da,
// test/geliştirme mock.go'da sağlar.
type Provider interface {
	Start(ctx context.Context, in StartInput) (StartResult, error)
	VerifyCallback(in CallbackInput) CallbackResult
	Refund(ctx context.Context, in RefundInput) error
}
