package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken(42, "cicekci", testSecret)
	require.NoError(t, err)

	claims, err := ParseToken(token, testSecret)

	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "cicekci", claims.Username)
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken(42, "cicekci", testSecret)
	require.NoError(t, err)

	_, err = ParseToken(token, "baska-bir-secret-uzunlugu-yeterli")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestParseToken_Garbage(t *testing.T) {
	_, err := ParseToken("bu-token-degil", testSecret)

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

// Süresi dolmuş token reddedilmeli. GenerateToken hep TokenTTL kullandığı
// için token'ı elle kurup geçmiş bir exp ile imzalıyoruz — ParseToken'ın
// exp claim'ini gerçekten kontrol ettiğini kanıtlar.
func TestParseToken_Expired(t *testing.T) {
	claims := Claims{
		UserID:   1,
		Username: "cicekci",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = ParseToken(expired, testSecret)

	require.ErrorIs(t, err, errorsx.ErrUnauthorized,
		"süresi dolmuş token kabul edilmemeli")
}
