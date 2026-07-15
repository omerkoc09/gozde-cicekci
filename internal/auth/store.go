package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// Store admin_users tablosu üzerinde CRUD işlemlerini yürütür.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// FindByUsername kullanıcı adına göre admin kullanıcısını getirir.
func (s *Store) FindByUsername(ctx context.Context, username string) (*AdminUser, error) {
	var u AdminUser
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM admin_users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin ara: %w", err)
	}
	return &u, nil
}

// Create yeni bir admin kullanıcısı kaydeder. passwordHash zaten hashlenmiş olmalı.
func (s *Store) Create(ctx context.Context, username, passwordHash string) (*AdminUser, error) {
	var u AdminUser
	err := s.pool.QueryRow(ctx,
		`INSERT INTO admin_users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, username, password_hash`,
		username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, errorsx.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("admin oluştur: %w", err)
	}
	return &u, nil
}
