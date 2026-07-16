package slider

// Slide ana sayfa slider'ında bir slayt. Görsel zorunlu — görselsiz slayt
// gösterilemez, o yüzden image_key hep dolu.
type Slide struct {
	ID        int64
	Title     string
	Subtitle  string
	ImageKey  string
	IsActive  bool
	SortOrder int
}

type CreateInput struct {
	Title     string
	Subtitle  string
	ImageKey  string
	IsActive  bool
	SortOrder int
}

// UpdateInput alanları pointer — nil olan alan değiştirilmez (PATCH semantiği).
// ImageKey burada yok: görsel değişimi ayrı uç üzerinden, çünkü eski dosyanın
// saklamadan silinmesi gerekiyor (bkz. Service.ReplaceImage).
type UpdateInput struct {
	Title     *string
	Subtitle  *string
	IsActive  *bool
	SortOrder *int
}
