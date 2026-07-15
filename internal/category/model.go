package category

type Axis string

const (
	AxisOccasion Axis = "occasion"
	AxisType     Axis = "type"
)

func (a Axis) Valid() bool {
	return a == AxisOccasion || a == AxisType
}

type Category struct {
	ID         int64
	Name       string
	Slug       string
	Axis       Axis
	IsActive   bool
	IsFeatured bool
	SortOrder  int
}

type CreateInput struct {
	Name       string
	Axis       Axis
	IsActive   bool
	IsFeatured bool
	SortOrder  int
}

// UpdateInput alanları pointer — nil olan alan değiştirilmez (PATCH semantiği).
// Axis alanı burada bilinçli olarak yok: eksen güncellenemez.
type UpdateInput struct {
	Name       *string
	IsActive   *bool
	IsFeatured *bool
	SortOrder  *int
}
