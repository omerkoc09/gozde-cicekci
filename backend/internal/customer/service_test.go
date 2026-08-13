package customer

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	return NewService(newTestStore(t), "test-jwt-secret-en-az-32-karakter-uzun")
}

func TestService_Register_TokenVeHashUretir(t *testing.T) {
	s := newTestService(t)
	tok, c, err := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.NotZero(t, c.ID)
	// Login ile doğrula (hash gerçekten kaydedilmiş mi)
	_, err = s.Login(context.Background(), "a@b.com", "sifre1234")
	require.NoError(t, err)
}

func TestService_Register_KisaSifreRed(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.Register(context.Background(), "a@b.com", "kisa", "Ali", "555")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Login_YanlisSifreUnauthorized(t *testing.T) {
	s := newTestService(t)
	_, _, _ = s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	_, err := s.Login(context.Background(), "a@b.com", "yanlissifre")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_Login_OlmayanKullaniciUnauthorized(t *testing.T) {
	s := newTestService(t)
	_, err := s.Login(context.Background(), "yok@b.com", "sifre1234")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_ChangePassword_MevcutSifreYanlisRed(t *testing.T) {
	s := newTestService(t)
	_, c, _ := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	err := s.ChangePassword(context.Background(), c.ID, "yanlis", "yenisifre1")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
	// doğru mevcut şifreyle geçer
	require.NoError(t, s.ChangePassword(context.Background(), c.ID, "sifre1234", "yenisifre1"))
}
