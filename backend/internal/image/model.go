package image

// ProductImage bir ürünün görseli. sort_order=0 olan kapak (spec §4.4).
type ProductImage struct {
	ID        int64
	ProductID int64
	ImageKey  string
	SortOrder int
}
