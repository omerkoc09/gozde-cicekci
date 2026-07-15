package category

import (
	"context"
	"fmt"
	"strings"

	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Category, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: kategori adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q (occasion veya type olmalı)",
			errorsx.ErrInvalidInput, in.Axis)
	}

	slug, err := s.uniqueSlug(ctx, product.Slugify(in.Name))
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, in, slug)
}

// uniqueSlug çakışma varsa -2, -3 ... ekler.
func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.store.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// Update kısmi günceller. Slug değişmez — kategori URL'leri sabit kalmalı (spec §4.2).
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*Category, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: kategori adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	return s.store.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) ListPublic(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis != nil && !axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q", errorsx.ErrInvalidInput, *axis)
	}
	return s.store.ListPublic(ctx, axis)
}

func (s *Service) ListFeatured(ctx context.Context) ([]Category, error) {
	return s.store.ListFeatured(ctx)
}

func (s *Service) ListAdmin(ctx context.Context) ([]Category, error) {
	return s.store.ListAdmin(ctx)
}

func (s *Service) GetPublicBySlug(ctx context.Context, slug string) (*Category, error) {
	return s.store.GetPublicBySlug(ctx, slug)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Category, error) {
	return s.store.GetByID(ctx, id)
}

// ProductCount silme öncesi uyarı için — "Bu kategoride N ürün var" (spec §4.1).
// Kategori yoksa ErrNotFound döner — count(*) aggregate olduğu için store
// tek başına bunu ayırt edemez, sayım 0 döner.
func (s *Service) ProductCount(ctx context.Context, id int64) (int, error) {
	if _, err := s.store.GetByID(ctx, id); err != nil {
		return 0, err
	}
	return s.store.ProductCount(ctx, id)
}
