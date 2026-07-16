package slider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// maxTitleLen slider başlığı ana sayfada tek satırda gösteriliyor —
// uzun metin tasarımı bozar.
const maxTitleLen = 120

type Service struct {
	store *Store
	img   *image.Service
}

func NewService(store *Store, img *image.Service) *Service {
	return &Service{store: store, img: img}
}

// Create slaytı görseliyle birlikte oluşturur. Görsel zorunlu.
//
// Atomiklik: önce dosya saklamaya yazılır, sonra DB'ye kayıt atılır.
// DB yazılamazsa dosya silinir — görselsiz slayt veya slaytsız dosya kalmaz
// (ürün görselleriyle aynı kural, spec §4.4).
func (s *Service) Create(ctx context.Context, in CreateInput, imageData []byte) (*Slide, error) {
	title := strings.TrimSpace(in.Title)
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	in.Title = title
	in.Subtitle = strings.TrimSpace(in.Subtitle)

	if len(imageData) == 0 {
		return nil, fmt.Errorf("%w: slayt görseli zorunlu", errorsx.ErrInvalidInput)
	}

	key, err := s.img.PutRaw(ctx, image.PrefixSlider, image.SliderSizes, imageData)
	if err != nil {
		return nil, err
	}
	in.ImageKey = key

	slide, err := s.store.Create(ctx, in)
	if err != nil {
		if delErr := s.img.DeleteRaw(ctx, image.PrefixSlider, key, image.SliderSizes); delErr != nil {
			log.Printf("DB hatası sonrası temizlik başarısız (key=%s): %v", key, delErr)
		}
		return nil, err
	}

	return slide, nil
}

// NextSortOrder yeni slaytın varsayılan sırası — mevcutların sonuna.
func (s *Service) NextSortOrder(ctx context.Context) (int, error) {
	max, err := s.store.MaxSortOrder(ctx)
	if err != nil {
		return 0, err
	}
	return max + 1, nil
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*Slide, error) {
	if in.Title != nil {
		trimmed := strings.TrimSpace(*in.Title)
		if err := validateTitle(trimmed); err != nil {
			return nil, err
		}
		in.Title = &trimmed
	}
	if in.Subtitle != nil {
		trimmed := strings.TrimSpace(*in.Subtitle)
		in.Subtitle = &trimmed
	}
	return s.store.Update(ctx, id, in)
}

// ReplaceImage slaytın görselini değiştirir. Yeni görsel yazılıp DB
// güncellendikten SONRA eski dosya silinir — sıra önemli: önce silseydik ve
// yeni yükleme patlasaydı slayt görselsiz kalırdı.
func (s *Service) ReplaceImage(ctx context.Context, id int64, imageData []byte) (*Slide, error) {
	current, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(imageData) == 0 {
		return nil, fmt.Errorf("%w: görsel boş", errorsx.ErrInvalidInput)
	}

	newKey, err := s.img.PutRaw(ctx, image.PrefixSlider, image.SliderSizes, imageData)
	if err != nil {
		return nil, err
	}

	slide, err := s.store.UpdateImageKey(ctx, id, newKey)
	if err != nil {
		if delErr := s.img.DeleteRaw(ctx, image.PrefixSlider, newKey, image.SliderSizes); delErr != nil {
			log.Printf("DB hatası sonrası temizlik başarısız (key=%s): %v", newKey, delErr)
		}
		return nil, err
	}

	// Eski dosya artık referanssız. Silinemezse yetim kalır ama site
	// bozulmaz — log'a düşer (spec §4.4).
	if err := s.img.DeleteRaw(ctx, image.PrefixSlider, current.ImageKey, image.SliderSizes); err != nil {
		log.Printf("yetim dosya: eski slayt görseli silinemedi (key=%s): %v", current.ImageKey, err)
	}

	return slide, nil
}

// Delete slaytı ve görselini siler. Key önce okunur — DB kaydı gidince
// hangi dosyanın silineceği öğrenilemez.
func (s *Service) Delete(ctx context.Context, id int64) error {
	slide, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.img.DeleteRaw(ctx, image.PrefixSlider, slide.ImageKey, image.SliderSizes); err != nil {
		log.Printf("yetim dosya: slayt görseli silinemedi (key=%s): %v", slide.ImageKey, err)
	}

	return nil
}

func (s *Service) ListPublic(ctx context.Context) ([]Slide, error) {
	return s.store.ListPublic(ctx)
}

func (s *Service) ListAdmin(ctx context.Context) ([]Slide, error) {
	return s.store.ListAdmin(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Slide, error) {
	return s.store.GetByID(ctx, id)
}

func validateTitle(title string) error {
	if title == "" {
		return fmt.Errorf("%w: slayt başlığı boş olamaz", errorsx.ErrInvalidInput)
	}
	if len([]rune(title)) > maxTitleLen {
		return fmt.Errorf("%w: slayt başlığı en fazla %d karakter olabilir",
			errorsx.ErrInvalidInput, maxTitleLen)
	}
	return nil
}
