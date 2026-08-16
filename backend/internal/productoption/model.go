package productoption

// Kind bir seçenek grubunun değerlerinin nasıl gösterileceğini söyler.
// Oluşturulduktan sonra DEĞİŞMEZ — color'dan text'e geçiş mevcut hex
// değerlerini anlamsız kılar (categories.axis ile aynı kural).
type Kind string

const (
	KindColor Kind = "color"
	KindText  Kind = "text"
)

func (k Kind) Valid() bool {
	return k == KindColor || k == KindText
}

// Group bir seçenek grubu — "Ambalaj Rengi". Values her zaman doludur
// (grup değerleriyle birlikte okunur), boş grup boş slice döner.
type Group struct {
	ID        int64
	Name      string
	Slug      string
	Kind      Kind
	SortOrder int
	IsActive  bool
	Values    []Value
}

// Value gruba bağlı tek seçenek — "Pembe" #F0A6CA.
// SwatchHex yalnızca KindColor gruplarda dolu.
type Value struct {
	ID        int64
	GroupID   int64
	Name      string
	SwatchHex string
	SortOrder int
	IsActive  bool
}

type CreateGroupInput struct {
	Name string
	Kind Kind
}

// UpdateGroupInput pointer alanlar PATCH semantiği — nil değişmez.
// Kind burada bilinçli olarak YOK: tip güncellenemez.
type UpdateGroupInput struct {
	Name     *string
	IsActive *bool
}

type CreateValueInput struct {
	GroupID   int64
	Name      string
	SwatchHex string
}

type UpdateValueInput struct {
	Name      *string
	SwatchHex *string
	IsActive  *bool
}

// ProductGroupLink ürün formundan gelen bağ — hangi grup bu üründe açık.
//
// Zorunluluk kavramı YOK (2026-08-16 kararı): müşteri sayfasında her grubun
// ilk değeri otomatik seçili geliyor, bu yüzden "seçilmeden geçilemez"
// kuralına gerek kalmadı. DB'deki product_option_groups.is_required sütunu
// yerinde duruyor ama hiçbir kod okumuyor/yazmıyor — INSERT varsayılanı
// (false) kullanılıyor.
type ProductGroupLink struct {
	GroupID int64
}

// ProductGroup ürüne açık bir grup.
type ProductGroup struct {
	Group
}

// GroupProduct bir seçenek grubunu kullanan ürün — panelde "bu grup nerede
// soruluyor" listesi için. Ürünün tamamı değil, listelemeye yetecek kadarı.
type GroupProduct struct {
	ID       int64
	Name     string
	IsActive bool
}
