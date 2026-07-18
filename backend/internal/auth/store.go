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

// List tüm admin kullanıcılarını kullanıcı adına göre sıralı döner.
func (s *Store) List(ctx context.Context) ([]AdminUser, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, username FROM admin_users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("admin listele: %w", err)
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, fmt.Errorf("admin listele: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin listele: %w", err)
	}
	return users, nil
}

// Count kayıtlı admin sayısını döner.
func (s *Store) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("admin say: %w", err)
	}
	return count, nil
}

// Delete bir admin kullanıcısını siler.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// UpdatePassword bir adminin şifre hash'ini günceller. passwordHash zaten hashlenmiş olmalı.
func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE admin_users SET password_hash = $2 WHERE id = $1`,
		id, passwordHash)
	if err != nil {
		return fmt.Errorf("şifre güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}
