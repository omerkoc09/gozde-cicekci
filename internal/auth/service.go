package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

// Service admin kimlik doğrulama iş kurallarını yürütür.
type Service struct {
	store     *Store
	jwtSecret string
}

func NewService(store *Store, jwtSecret string) *Service {
	return &Service{store: store, jwtSecret: jwtSecret}
}

// CreateAdmin yeni bir admin kullanıcısı oluşturur. cmd/seed tarafından çağrılır.
func (s *Service) CreateAdmin(ctx context.Context, username, password string) error {
	if username == "" {
		return fmt.Errorf("%w: kullanıcı adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("şifre hashle: %w", err)
	}

	if _, err := s.store.Create(ctx, username, string(hash)); err != nil {
		return err
	}
	return nil
}

// Login kullanıcı adı ve şifreyi doğrular, başarılıysa JWT token döner.
// Kullanıcı yok ile şifre yanlış aynı hatayı döner — bilgi sızdırmamak için.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.store.FindByUsername(ctx, username)
	if errors.Is(err, errorsx.ErrNotFound) {
		return "", errorsx.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errorsx.ErrUnauthorized
	}

	return GenerateToken(user.ID, user.Username, s.jwtSecret)
}
