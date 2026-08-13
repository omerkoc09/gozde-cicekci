package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// beklenenCallbackHash test yardımcı — dokümandaki formülü bağımsız üretir.
func beklenenCallbackHash(oid, salt, status, total, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(oid + salt + status + total))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestPayTR_VerifyCallback_DogruHashKabul(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	hash := beklenenCallbackHash("oid1", "salt", "success", "18500", "key")

	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "success", TotalAmount: "18500", Hash: hash,
	})
	if !res.OK {
		t.Fatal("doğru hash + success için OK bekleniyordu")
	}
}

func TestPayTR_VerifyCallback_YanlisHashRed(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "success", TotalAmount: "18500", Hash: "SAHTE",
	})
	if res.OK {
		t.Fatal("yanlış hash reddedilmeliydi — bedava sipariş riski")
	}
}

func TestPayTR_VerifyCallback_FailedStatusOKDegil(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	hash := beklenenCallbackHash("oid1", "salt", "failed", "0", "key")
	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "failed", TotalAmount: "0", Hash: hash,
	})
	if res.OK {
		t.Fatal("failed status için OK olmamalı (hash doğru olsa bile)")
	}
}

func TestKurusFromDecimal(t *testing.T) {
	cases := map[string]int64{
		"1850.00": 185000,
		"1850.50": 185050,
		"0.01":    1,
		"1234.56": 123456,
	}
	for in, want := range cases {
		d := decimal.RequireFromString(in)
		if got := KurusFromDecimal(d); got != want {
			t.Errorf("KurusFromDecimal(%s) = %d, beklenen %d", in, got, want)
		}
	}
}

// beklenenTokenHash Start için dokümandaki formülü bağımsız üretir:
// merchant_id + user_ip + merchant_oid + email + payment_amount +
// user_basket + no_installment + max_installment + currency + test_mode + merchant_salt
func beklenenTokenHash(merchantID, userIP, oid, email, amount, basket, noInst, maxInst, currency, testMode, salt, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(merchantID + userIP + oid + email + amount + basket + noInst + maxInst + currency + testMode + salt))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// beklenenRefundHash Refund için dokümandaki formülü bağımsız üretir:
// merchant_id + merchant_oid + return_amount + merchant_salt
func beklenenRefundHash(merchantID, oid, returnAmount, salt, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(merchantID + oid + returnAmount + salt))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestPayTR_Start_TokenAlir(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("form parse edilemedi: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "token": "abc"})
	}))
	defer srv.Close()

	p := NewPayTR(PayTRConfig{
		MerchantID: "m1", MerchantKey: "key", MerchantSalt: "salt", TestMode: true,
	})
	p.tokenURL = srv.URL

	in := StartInput{
		MerchantOID: "oid-123",
		UserIP:      "1.2.3.4",
		Email:       "a@b.com",
		AmountKurus: 375000,
		Basket:      []BasketItem{{Name: "51 Gül", PriceKurus: 185000, Quantity: 1}},
		OkURL:       "https://example.com/ok",
		FailURL:     "https://example.com/fail",
	}

	res, err := p.Start(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, "abc", res.Token)

	require.NotNil(t, gotForm)
	assert.Equal(t, "m1", gotForm.Get("merchant_id"))
	assert.Equal(t, "oid-123", gotForm.Get("merchant_oid"))
	assert.Equal(t, "375000", gotForm.Get("payment_amount"))
	assert.NotEmpty(t, gotForm.Get("user_basket"))
	assert.NotEmpty(t, gotForm.Get("paytr_token"))

	wantBasket := encodeBasket(in.Basket)
	wantHash := beklenenTokenHash("m1", "1.2.3.4", "oid-123", "a@b.com", "375000",
		wantBasket, "0", "0", "TL", "1", "salt", "key")
	assert.Equal(t, wantHash, gotForm.Get("paytr_token"), "paytr_token hash string sırası dokümana uymalı")
}

func TestPayTR_Start_HataReddeder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "reason": "gecersiz istek"})
	}))
	defer srv.Close()

	p := NewPayTR(PayTRConfig{MerchantID: "m1", MerchantKey: "key", MerchantSalt: "salt"})
	p.tokenURL = srv.URL

	_, err := p.Start(context.Background(), StartInput{
		MerchantOID: "oid-1", UserIP: "1.2.3.4", Email: "a@b.com", AmountKurus: 1000,
	})
	assert.Error(t, err)
}

func TestPayTR_Refund_BasariliVeHash(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("form parse edilemedi: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer srv.Close()

	p := NewPayTR(PayTRConfig{MerchantID: "m1", MerchantKey: "key", MerchantSalt: "salt"})
	p.refundURL = srv.URL

	err := p.Refund(context.Background(), RefundInput{
		MerchantOID:       "oid-9",
		ReturnAmountKurus: 64900,
	})
	require.NoError(t, err)

	require.NotNil(t, gotForm)
	assert.Equal(t, "m1", gotForm.Get("merchant_id"))
	assert.Equal(t, "oid-9", gotForm.Get("merchant_oid"))
	assert.Equal(t, "649.00", gotForm.Get("return_amount"), "kuruş→TL ondalık dönüşümü")

	wantHash := beklenenRefundHash("m1", "oid-9", "649.00", "salt", "key")
	assert.Equal(t, wantHash, gotForm.Get("paytr_token"), "iade hash string sırası dokümana uymalı")
}

func TestPayTR_Refund_HataReddeder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "err_msg": "iade reddedildi"})
	}))
	defer srv.Close()

	p := NewPayTR(PayTRConfig{MerchantID: "m1", MerchantKey: "key", MerchantSalt: "salt"})
	p.refundURL = srv.URL

	err := p.Refund(context.Background(), RefundInput{MerchantOID: "oid-9", ReturnAmountKurus: 1000})
	assert.Error(t, err)
}
