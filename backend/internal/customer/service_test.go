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

// TestService_Register_UzunSifreGecersizGirdiDoner I5 regresyon testi.
// bcrypt.GenerateFromPassword 72 bayt üzerinde ErrPasswordTooLong ile sert
// başarısız olur; bu hata errorsx sentinel'lerinden hiçbirine uymadığı için
// api.WriteError'ın default dalı 500 "Sunucu hatası" döndürüyordu. Register
// artık uzunluğu erkenden kontrol edip ErrInvalidInput dönmeli (400).
func TestService_Register_UzunSifreGecersizGirdiDoner(t *testing.T) {
	s := newTestService(t)
	uzunSifre := make([]byte, 73)
	for i := range uzunSifre {
		uzunSifre[i] = 'a'
	}
	_, _, err := s.Register(context.Background(), "uzun@b.com", string(uzunSifre), "Ali", "555")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput,
		"73 baytlık şifre 500 değil ErrInvalidInput dönmeli")
}

// TestService_ChangePassword_UzunSifreGecersizGirdiDoner Register ile aynı
// bcrypt sınırı ChangePassword'da da uygulanmalı (I5, ikinci çağrı yeri).
func TestService_ChangePassword_UzunSifreGecersizGirdiDoner(t *testing.T) {
	s := newTestService(t)
	_, c, err := s.Register(context.Background(), "uzun2@b.com", "sifre1234", "Ali", "555")
	require.NoError(t, err)

	uzunSifre := make([]byte, 73)
	for i := range uzunSifre {
		uzunSifre[i] = 'b'
	}
	err = s.ChangePassword(context.Background(), c.ID, "sifre1234", string(uzunSifre))
	require.ErrorIs(t, err, errorsx.ErrInvalidInput,
		"73 baytlık yeni şifre 500 değil ErrInvalidInput dönmeli")
}
