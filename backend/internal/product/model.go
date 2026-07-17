package product

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	Price       decimal.Decimal
	IsActive    bool
	IsFeatured  bool
	CategoryIDs []int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateInput struct {
	Name        string
	Description string
	Price       decimal.Decimal
	IsActive    bool
	IsFeatured  bool
	CategoryIDs []int64
}

// UpdateInput pointer alanlar PATCH semantiği — nil değişmez.
// CategoryIDs nil ise kategoriler değişmez; boş slice ise hepsi kaldırılır.
type UpdateInput struct {
	Name        *string
	Description *string
	Price       *decimal.Decimal
	IsActive    *bool
	IsFeatured  *bool
	CategoryIDs []int64
}

// Filter iki eksenli filtreleme. İkisi de doluysa AND — her iki koşula da
// uyan ürünler (spec §5.6).
type Filter struct {
	OccasionSlug *string
	TypeSlug     *string

	// Query doluysa ürün adında veya bağlı kategori adında arama yapar
	// (case-insensitive substring), diğer koşullarla AND birleşir.
	Query *string

	// FeaturedOnly true ise yalnızca öne çıkan ürünler döner — ana sayfa
	// vitrini bunu kullanıyor. false ise öne çıkma durumu filtrelemez.
	FeaturedOnly bool

	Limit  int
	Offset int
}
