package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/shopspring/decimal"
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
