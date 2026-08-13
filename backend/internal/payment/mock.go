package payment

import "context"

// MockProvider gerçek PayTR anahtarları olmadan geliştirme/test için.
// Start sabit bir token döner; VerifyCallback status=="success" ise OK der
// (hash kontrolü yok — mock). Refund her zaman başarılı.
type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (m *MockProvider) Start(_ context.Context, in StartInput) (StartResult, error) {
	return StartResult{Token: "mock-token-" + in.MerchantOID}, nil
}

func (m *MockProvider) VerifyCallback(in CallbackInput) CallbackResult {
	return CallbackResult{OK: in.Status == "success", MerchantOID: in.MerchantOID}
}

func (m *MockProvider) Refund(_ context.Context, _ RefundInput) error {
	return nil
}
