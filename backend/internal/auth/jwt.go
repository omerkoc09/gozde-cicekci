package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// TokenTTL üretilen JWT token'ların geçerlilik süresidir.
const TokenTTL = 7 * 24 * time.Hour

// claimType admin token'ını müşteri token'ından ayıran değer. Her iki token
// türü de AYNI cfg.JWTSecret ile imzalanıyor (bkz. main.go), yani imza
// tek başına ayırt edici değil — typ claim'i olmadan bu ayrım yalnızca
// claim alan adlarının (uid/usr vs cid) tesadüfen çakışmamasına dayanır.
// Biri "Type" alanı eklerse ya da "cid"i "uid" yaparsa sınır sessizce
// çöker. typ zorunluluğu bunu yapısal hale getirir.
const claimType = "admin"

// Claims JWT içinde taşınan admin kullanıcı bilgileridir.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	Type     string `json:"typ"`
	jwt.RegisteredClaims
}

// GenerateToken verilen kullanıcı bilgileri için imzalı bir JWT üretir.
func GenerateToken(userID int64, username, secret string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Type:     claimType,
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
// Ayrıca typ claim'i doluyken "admin" değilse reddeder — bu, customer_token
// (typ:"customer") bir şekilde cicekci_token cookie'sine konsa bile admin
// uçlarına erişemeyeceğini garanti eder (customer.ParseToken'ın tersini
// zaten yaptığı kontrolün simetriği). Eski admin token'ları typ taşımadığı
// (Type == "") için geriye dönük uyumluluk BOZULMAZ.
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
	if !ok || (claims.Type != "" && claims.Type != claimType) {
		return nil, errorsx.ErrUnauthorized
	}
	return claims, nil
}
