package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// TokenTTL üretilen JWT token'ların geçerlilik süresidir.
const TokenTTL = 7 * 24 * time.Hour

// Claims JWT içinde taşınan admin kullanıcı bilgileridir.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

// GenerateToken verilen kullanıcı bilgileri için imzalı bir JWT üretir.
func GenerateToken(userID int64, username, secret string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
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

// ParseToken bir JWT'yi doğrular ve içindeki claims'i döner.
// İmza yöntemi HMAC değilse reddeder ("alg: none" saldırısına karşı korunma).
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
	if !ok {
		return nil, errorsx.ErrUnauthorized
	}
	return claims, nil
}
