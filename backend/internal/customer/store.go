package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create yeni müşteri kaydeder. passwordHash zaten hashlenmiş olmalı.
// Email çakışması (UNIQUE) → ErrConflict.
func (s *Store) Create(ctx context.Context, email, passwordHash, name, phone string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, phone)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, email, name, phone, password_hash`,
		email, passwordHash, name, phone,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, errorsx.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri oluştur: %w", err)
	}
	return &cst, nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash FROM customers WHERE email = $1`,
		email,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri ara: %w", err)
	}
	return &cst, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash FROM customers WHERE id = $1`,
		id,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri getir: %w", err)
	}
	return &cst, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, name, phone string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET name=$2, phone=$3, updated_at=now() WHERE id=$1`,
		id, name, phone)
	if err != nil {
		return fmt.Errorf("profil güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET password_hash=$2, updated_at=now() WHERE id=$1`,
		id, passwordHash)
	if err != nil {
		return fmt.Errorf("şifre güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}
