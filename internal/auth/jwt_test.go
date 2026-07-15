package auth

import (
	"testing"

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
