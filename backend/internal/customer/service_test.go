package customer

import (
	"context"
	"strings"
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	return NewService(newTestStore(t), "test-jwt-secret-en-az-32-karakter-uzun")
}

func TestService_Register_TokenVeHashUretir(t *testing.T) {
	s := newTestService(t)
	tok, c, err := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "5551112233")
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.NotZero(t, c.ID)
	// Login ile doğrula (hash gerçekten kaydedilmiş mi)
	_, err = s.Login(context.Background(), "a@b.com", "sifre1234")
	require.NoError(t, err)
}

func TestService_Register_KisaSifreRed(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.Register(context.Background(), "a@b.com", "kisa", "Ali", "5551112233")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// TestService_Register_GecersizTelefonRed kayıtta telefon doğrulaması.
// Önceden hiçbir kontrol yoktu: "asdasd" ile hesap açılabiliyordu ve
// teslimat için müşteriye ulaşılamıyordu.
func TestService_Register_GecersizTelefonRed(t *testing.T) {
	s := newTestService(t)
	gecersiz := []string{"asdasd", "555", "2121112233", "55511122334", ""}
	for _, tel := range gecersiz {
		_, _, err := s.Register(context.Background(), "tel@b.com", "sifre1234", "Ali", tel)
		require.ErrorIs(t, err, errorsx.ErrInvalidInput, "telefon %q reddedilmeliydi", tel)
	}
}

// TestService_Register_TelefonNormalize kullanıcı hangi biçimde yazarsa
// yazsın veritabanına tek biçimde ("5551112233") yazılmalı.
func TestService_Register_TelefonNormalize(t *testing.T) {
	s := newTestService(t)
	_, c, err := s.Register(context.Background(), "norm@b.com", "sifre1234", "Ali", "+90 555 111 22 33")
	require.NoError(t, err)
	require.Equal(t, "5551112233", c.Phone)
}

// TestService_UpdateProfile_GecersizTelefonRed aynı kural profil
// güncellemede de geçerli olmalı — yoksa kullanıcı kayıttan sonra
// numarayı bozabilirdi.
func TestService_UpdateProfile_GecersizTelefonRed(t *testing.T) {
	s := newTestService(t)
	_, c, err := s.Register(context.Background(), "prof@b.com", "sifre1234", "Ali", "5551112233")
	require.NoError(t, err)

	_, err = s.UpdateProfile(context.Background(), c.ID, "Ali", "asdasd")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput)

	// Geçerli numara normalize edilerek kaydedilir.
	got, err := s.UpdateProfile(context.Background(), c.ID, "Ali", "0555 999 88 77")
	require.NoError(t, err)
	require.Equal(t, "5559998877", got.Phone)
}

// TestService_Register_GecersizEpostaRed (M2) — eski kontrol
// strings.Contains(email,"@") && len>=3 idi ve "a@", "@b" gibi çöpleri
// kabul ediyordu; yani API, kayıt formundaki type="email" alanının
// reddettiği değerleri kabul ediyordu.
func TestService_Register_GecersizEpostaRed(t *testing.T) {
	s := newTestService(t)
	gecersiz := []string{"a@", "@b", "@ ", "duz-metin", "", "a@b@c.com", "Ad <a@b.com>"}
	for _, e := range gecersiz {
		_, _, err := s.Register(context.Background(), e, "sifre1234", "Ali", "5551112233")
		require.ErrorIs(t, err, errorsx.ErrInvalidInput, "e-posta %q reddedilmeliydi", e)
	}
}

// TestService_Register_UzunAlanlarRed (M1) — kolonlar düz TEXT olduğu için
// sınır yoksa kimlik doğrulaması gerektirmeyen kayıt ucuna megabaytlık
// değerler yazılabiliyordu.
func TestService_Register_UzunAlanlarRed(t *testing.T) {
	s := newTestService(t)

	uzunEposta := strings.Repeat("a", 250) + "@ornek.com" // 260 > 254
	_, _, err := s.Register(context.Background(), uzunEposta, "sifre1234", "Ali", "5551112233")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput, "254 karakterden uzun e-posta reddedilmeli")

	uzunAd := strings.Repeat("a", 121) // 121 > 120
	_, _, err = s.Register(context.Background(), "uzunad@b.com", "sifre1234", uzunAd, "5551112233")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput, "120 karakterden uzun ad reddedilmeli")
}

// TestService_UpdateProfile_UzunAdRed sınır profil güncellemede de geçerli.
func TestService_UpdateProfile_UzunAdRed(t *testing.T) {
	s := newTestService(t)
	_, c, err := s.Register(context.Background(), "profiluzun@b.com", "sifre1234", "Ali", "5551112233")
	require.NoError(t, err)

	_, err = s.UpdateProfile(context.Background(), c.ID, strings.Repeat("a", 121), "5551112233")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Login_YanlisSifreUnauthorized(t *testing.T) {
	s := newTestService(t)
	_, _, _ = s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "5551112233")
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
	_, c, _ := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "5551112233")
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
	_, _, err := s.Register(context.Background(), "uzun@b.com", string(uzunSifre), "Ali", "5551112233")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput,
		"73 baytlık şifre 500 değil ErrInvalidInput dönmeli")
}

// TestService_ChangePassword_UzunSifreGecersizGirdiDoner Register ile aynı
// bcrypt sınırı ChangePassword'da da uygulanmalı (I5, ikinci çağrı yeri).
func TestService_ChangePassword_UzunSifreGecersizGirdiDoner(t *testing.T) {
	s := newTestService(t)
	_, c, err := s.Register(context.Background(), "uzun2@b.com", "sifre1234", "Ali", "5551112233")
	require.NoError(t, err)

	uzunSifre := make([]byte, 73)
	for i := range uzunSifre {
		uzunSifre[i] = 'b'
	}
	err = s.ChangePassword(context.Background(), c.ID, "sifre1234", string(uzunSifre))
	require.ErrorIs(t, err, errorsx.ErrInvalidInput,
		"73 baytlık yeni şifre 500 değil ErrInvalidInput dönmeli")
}
