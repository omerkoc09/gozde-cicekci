package category

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Service struct {
	store *Store
	img   *image.Service
}

// NewService — img nil olabilir; yalnızca görsel uçları (ReplaceImage,
// DeleteImage) onu kullanıyor, listeleme/CRUD görselsiz de çalışır.
func NewService(store *Store, img *image.Service) *Service {
	return &Service{store: store, img: img}
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

// ReplaceImage kategori kart görselini değiştirir. Sıra slider'daki ile aynı:
// önce yeni dosya yazılır ve DB güncellenir, SONRA eskisi silinir — tersi
// olsaydı yeni yükleme patladığında kategori görselsiz kalırdı.
func (s *Service) ReplaceImage(ctx context.Context, id int64, data []byte) (*Category, error) {
	current, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("%w: görsel boş", errorsx.ErrInvalidInput)
	}

	newKey, err := s.img.PutRaw(ctx, image.PrefixCategories, image.CategorySizes, data)
	if err != nil {
		return nil, err
	}

	cat, err := s.store.UpdateImageKey(ctx, id, newKey)
	if err != nil {
		if delErr := s.img.DeleteRaw(ctx, image.PrefixCategories, newKey, image.CategorySizes); delErr != nil {
			log.Printf("DB hatası sonrası temizlik başarısız (key=%s): %v", newKey, delErr)
		}
		return nil, err
	}

	if current.ImageKey != "" {
		if err := s.img.DeleteRaw(ctx, image.PrefixCategories, current.ImageKey, image.CategorySizes); err != nil {
			log.Printf("yetim dosya: eski kategori görseli silinemedi (key=%s): %v", current.ImageKey, err)
		}
	}

	return cat, nil
}

// DeleteImage kart görselini kaldırır — kategori yedek görsele döner.
func (s *Service) DeleteImage(ctx context.Context, id int64) (*Category, error) {
	current, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.ImageKey == "" {
		return current, nil // zaten görselsiz — idempotent
	}

	cat, err := s.store.UpdateImageKey(ctx, id, "")
	if err != nil {
		return nil, err
	}

	if err := s.img.DeleteRaw(ctx, image.PrefixCategories, current.ImageKey, image.CategorySizes); err != nil {
		log.Printf("yetim dosya: kategori görseli silinemedi (key=%s): %v", current.ImageKey, err)
	}
	return cat, nil
}

// Delete kategoriyi siler. Görsel key'i önce okunur — DB kaydı gidince
// hangi dosyanın silineceği öğrenilemez.
func (s *Service) Delete(ctx context.Context, id int64) error {
	cat, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	if cat.ImageKey != "" && s.img != nil {
		if err := s.img.DeleteRaw(ctx, image.PrefixCategories, cat.ImageKey, image.CategorySizes); err != nil {
			log.Printf("yetim dosya: kategori görseli silinemedi (key=%s): %v", cat.ImageKey, err)
		}
	}
	return nil
}

func (s *Service) ListPublic(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis != nil && !axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q", errorsx.ErrInvalidInput, *axis)
	}
	return s.store.ListPublic(ctx, axis)
}

// ListFeatured axis nil ise iki eksenden de öne çıkanları döner.
func (s *Service) ListFeatured(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis != nil && !axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q", errorsx.ErrInvalidInput, *axis)
	}
	return s.store.ListFeatured(ctx, axis)
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
