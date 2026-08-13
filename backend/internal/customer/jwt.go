package customer

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// TokenTTL üretilen müşteri JWT'lerinin geçerlilik süresi.
const TokenTTL = 7 * 24 * time.Hour

// claimType müşteri token'ını admin token'ından ayıran değer.
// Middleware yalnızca bu değeri taşıyan token'ları kabul eder.
const claimType = "customer"

// Claims müşteri JWT'sinde taşınan bilgiler. Type alanı, admin token'ının
// yanlışlıkla müşteri ucuna geçmesini (veya tersini) engeller.
type Claims struct {
	CustomerID int64  `json:"cid"`
	Type       string `json:"typ"`
	jwt.RegisteredClaims
}

func GenerateToken(customerID int64, secret string) (string, error) {
	claims := Claims{
		CustomerID: customerID,
		Type:       claimType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("token imzala: %w", err)
	}
	return signed, nil
}

// ParseToken JWT'yi doğrular. HMAC dışı imza yöntemini ("alg: none" saldırısı)
// ve Type != "customer" olan token'ı (admin token'ı) reddeder.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen imza yöntemi: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errorsx.ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.Type != claimType {
		return nil, errorsx.ErrUnauthorized
	}
	return claims, nil
}
