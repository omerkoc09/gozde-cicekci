// Package errorsx uygulama genelinde kullanılan ortak, sentinel hataları tanımlar.
// Service katmanı bu hataları döner, API katmanı bunları HTTP durum kodlarına eşler.
package errorsx

import "errors"

var (
	ErrNotFound     = errors.New("kayıt bulunamadı")
	ErrInvalidInput = errors.New("geçersiz girdi")
	ErrUnauthorized = errors.New("yetkisiz")
	ErrConflict     = errors.New("çakışma")
)
