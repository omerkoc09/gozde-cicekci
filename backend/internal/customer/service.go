package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

type Service struct {
	store     *Store
	jwtSecret string
}

func NewService(store *Store, jwtSecret string) *Service {
	return &Service{store: store, jwtSecret: jwtSecret}
}

// Register yeni müşteri hesabı açar ve otomatik giriş token'ı döner.
func (s *Service) Register(ctx context.Context, email, password, name, phone string) (string, *Customer, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	if !strings.Contains(email, "@") || len(email) < 3 {
		return "", nil, fmt.Errorf("%w: geçerli bir e-posta girin", errorsx.ErrInvalidInput)
	}
	if len(password) < minPasswordLength {
		return "", nil, fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}
	if name == "" {
		return "", nil, fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	}
	if phone == "" {
		return "", nil, fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("şifre hashle: %w", err)
	}

	cst, err := s.store.Create(ctx, email, string(hash), name, phone)
	if errors.Is(err, errorsx.ErrConflict) {
		return "", nil, fmt.Errorf("%w: bu e-posta ile hesap var, giriş yapın", errorsx.ErrConflict)
	}
	if err != nil {
		return "", nil, err
	}

	token, err := GenerateToken(cst.ID, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}
	return token, cst, nil
}

// Login e-posta+şifre doğrular. Kullanıcı yok ile şifre yanlış aynı hatayı
// döner — bilgi sızdırmamak için (admin auth deseni).
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	cst, err := s.store.FindByEmail(ctx, email)
	if errors.Is(err, errorsx.ErrNotFound) {
		return "", errorsx.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cst.PasswordHash), []byte(password)); err != nil {
		return "", errorsx.ErrUnauthorized
	}
	return GenerateToken(cst.ID, s.jwtSecret)
}

func (s *Service) Get(ctx context.Context, id int64) (*Customer, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id int64, name, phone string) (*Customer, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	if name == "" {
		return nil, fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	}
	if phone == "" {
		return nil, fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	}
	if err := s.store.UpdateProfile(ctx, id, name, phone); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// ChangePassword mevcut şifreyi doğrulamadan değiştirmez (cookie çalınırsa
// şifre değiştirilemesin).
func (s *Service) ChangePassword(ctx context.Context, id int64, currentPassword, newPassword string) error {
	cst, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cst.PasswordHash), []byte(currentPassword)); err != nil {
		return errorsx.ErrUnauthorized
	}
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("şifre hashle: %w", err)
	}
	return s.store.UpdatePassword(ctx, id, string(hash))
}
