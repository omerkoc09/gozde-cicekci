package payment

import (
	"context"
	"testing"
)

func TestMockProvider_Start_TokenDoner(t *testing.T) {
	m := NewMockProvider()
	res, err := m.Start(context.Background(), StartInput{MerchantOID: "abc"})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if res.Token != "mock-token-abc" {
		t.Fatalf("token = %q", res.Token)
	}
}

func TestMockProvider_VerifyCallback_SuccessOK(t *testing.T) {
	m := NewMockProvider()
	if !m.VerifyCallback(CallbackInput{MerchantOID: "abc", Status: "success"}).OK {
		t.Fatal("success için OK bekleniyordu")
	}
	if m.VerifyCallback(CallbackInput{MerchantOID: "abc", Status: "failed"}).OK {
		t.Fatal("failed için OK olmamalı")
	}
}
