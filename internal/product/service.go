package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/omerkoc/cicekci/pkg/errorsx"
)

const (
	defaultLimit = 24
	maxLimit     = 100
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Product, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: ürün adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if in.Price.IsNegative() {
		return nil, fmt.Errorf("%w: fiyat negatif olamaz", errorsx.ErrInvalidInput)
	}

	slug, err := s.uniqueSlug(ctx, Slugify(in.Name))
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, in, slug)
}

// uniqueSlug çakışma varsa -2, -3 ... ekler. Eski slug'lar da çakışma sayılır.
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

// Update ürünü günceller. İsim değişirse yeni slug üretilir ve eskisi
// is_current=false olarak saklanır — eski linkler 301 ile yaşamaya devam
// eder (spec §4.2).
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*Product, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: ürün adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	if in.Price != nil && in.Price.IsNegative() {
		return nil, fmt.Errorf("%w: fiyat negatif olamaz", errorsx.ErrInvalidInput)
	}

	current, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.store.Update(ctx, id, in)
	if err != nil {
		return nil, err
	}

	// İsim değiştiyse ve yeni slug farklıysa slug geçmişini güncelle.
	if in.Name != nil && *in.Name != current.Name {
		newSlugBase := Slugify(*in.Name)
		if newSlugBase != current.Slug {
			newSlug, err := s.uniqueSlug(ctx, newSlugBase)
			if err != nil {
				return nil, err
			}
			if err := s.store.AddSlug(ctx, id, newSlug); err != nil {
				return nil, err
			}
			updated, err = s.store.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
		}
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

// GetPublicBySlug ürünü slug'dan bulur. Slug eskiyse redirectTo dolu döner
// ve handler 301 yapmalıdır. Pasif ürün ErrNotFound döner.
func (s *Service) GetPublicBySlug(ctx context.Context, slug string) (*Product, string, error) {
	productID, isCurrent, err := s.store.FindSlug(ctx, slug)
	if err != nil {
		return nil, "", err
	}

	p, err := s.store.GetPublicByID(ctx, productID)
	if err != nil {
		return nil, "", err
	}

	if !isCurrent {
		return p, p.Slug, nil
	}
	return p, "", nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Product, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) ListPublic(ctx context.Context, f Filter) ([]Product, error) {
	f.Limit = normalizeLimit(f.Limit)
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.store.ListPublic(ctx, f)
}

func (s *Service) ListAdmin(ctx context.Context, limit, offset int) ([]Product, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		offset = 0
	}
	return s.store.ListAdmin(ctx, limit, offset)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
