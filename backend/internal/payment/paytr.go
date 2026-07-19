package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	tokenURL  = "https://www.paytr.com/odeme/api/get-token"
	refundURL = "https://www.paytr.com/odeme/iade"
)

type PayTRConfig struct {
	MerchantID   string
	MerchantKey  string
	MerchantSalt string
	TestMode     bool
	HTTPClient   *http.Client
}

type PayTRProvider struct {
	cfg    PayTRConfig
	client *http.Client

	// tokenURL/refundURL normalde sabit PayTR uç noktaları. Alan olarak
	// tutulmalarının tek sebebi test edilebilirlik — testte httptest.Server
	// adresine yönlendirilirler. NewPayTR dışında değiştirilmemeli.
	tokenURL  string
	refundURL string
}

func NewPayTR(cfg PayTRConfig) *PayTRProvider {
	c := cfg.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 20 * time.Second}
	}
	return &PayTRProvider{cfg: cfg, client: c, tokenURL: tokenURL, refundURL: refundURL}
}

// KurusFromDecimal tutarı kuruşa çevirir (× 100, tam sayı). PayTR kuruş bekler.
func KurusFromDecimal(d decimal.Decimal) int64 {
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func (p *PayTRProvider) hmacBase64(s string) string {
	mac := hmac.New(sha256.New, []byte(p.cfg.MerchantKey))
	mac.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (p *PayTRProvider) testModeStr() string {
	if p.cfg.TestMode {
		return "1"
	}
	return "0"
}

// encodeBasket PayTR user_basket formatı: [["ad","fiyat_kurus","adet"],...] → base64(JSON).
func encodeBasket(items []BasketItem) string {
	rows := make([][]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, []any{it.Name, strconv.FormatInt(it.PriceKurus, 10), it.Quantity})
	}
	b, _ := json.Marshal(rows)
	return base64.StdEncoding.EncodeToString(b)
}

func (p *PayTRProvider) Start(ctx context.Context, in StartInput) (StartResult, error) {
	basket := encodeBasket(in.Basket)
	amount := strconv.FormatInt(in.AmountKurus, 10)
	noInstallment := "0"
	maxInstallment := "0"
	currency := "TL"

	// Token hash string (doküman sırası):
	// merchant_id + user_ip + merchant_oid + email + payment_amount +
	// user_basket + no_installment + max_installment + currency + test_mode + merchant_salt
	hashStr := p.cfg.MerchantID + in.UserIP + in.MerchantOID + in.Email + amount +
		basket + noInstallment + maxInstallment + currency + p.testModeStr() + p.cfg.MerchantSalt
	token := p.hmacBase64(hashStr)

	form := url.Values{
		"merchant_id":       {p.cfg.MerchantID},
		"user_ip":           {in.UserIP},
		"merchant_oid":      {in.MerchantOID},
		"email":             {in.Email},
		"payment_amount":    {amount},
		"paytr_token":       {token},
		"user_basket":       {basket},
		"no_installment":    {noInstallment},
		"max_installment":   {maxInstallment},
		"currency":          {currency},
		"test_mode":         {p.testModeStr()},
		"merchant_ok_url":   {in.OkURL},
		"merchant_fail_url": {in.FailURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return StartResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return StartResult{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
		Token  string `json:"token"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return StartResult{}, fmt.Errorf("paytr yanıtı okunamadı: %w", err)
	}
	if out.Status != "success" {
		return StartResult{}, fmt.Errorf("paytr token reddetti: %s", out.Reason)
	}

	return StartResult{Token: out.Token}, nil
}

func (p *PayTRProvider) VerifyCallback(in CallbackInput) CallbackResult {
	// Callback hash string: merchant_oid + merchant_salt + status + total_amount
	expected := p.hmacBase64(in.MerchantOID + p.cfg.MerchantSalt + in.Status + in.TotalAmount)
	if !hmac.Equal([]byte(expected), []byte(in.Hash)) {
		return CallbackResult{OK: false, MerchantOID: in.MerchantOID}
	}
	return CallbackResult{OK: in.Status == "success", MerchantOID: in.MerchantOID}
}

func (p *PayTRProvider) Refund(ctx context.Context, in RefundInput) error {
	// return_amount tam TL değil kuruş değil — PayTR iade "TL cinsinden ondalık"
	// bekler (ör. "18.50"). Kuruşu 100'e bölerek string üretiyoruz.
	amount := decimal.New(in.ReturnAmountKurus, -2).StringFixed(2)

	// İade hash string: merchant_id + merchant_oid + return_amount + merchant_salt
	token := p.hmacBase64(p.cfg.MerchantID + in.MerchantOID + amount + p.cfg.MerchantSalt)

	form := url.Values{
		"merchant_id":   {p.cfg.MerchantID},
		"merchant_oid":  {in.MerchantOID},
		"return_amount": {amount},
		"paytr_token":   {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.refundURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
		ErrMsg string `json:"err_msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("paytr iade yanıtı okunamadı: %w", err)
	}
	if out.Status != "success" {
		return fmt.Errorf("paytr iade reddetti: %s", out.ErrMsg)
	}
	return nil
}
