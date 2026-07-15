package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basit", "Buket", "buket"},
		{"bosluk", "51 Gül Buket", "51-gul-buket"},
		{"turkce karakterler", "Çiçek Şöleni Güzel", "cicek-soleni-guzel"},
		{"buyuk I", "İstanbul Lalesi", "istanbul-lalesi"},
		{"noktali i", "Ilık Bahar", "ilik-bahar"},
		{"noktalama", "Gül & Papatya (Özel!)", "gul-papatya-ozel"},
		{"coklu bosluk", "Kırmızı   Gül", "kirmizi-gul"},
		{"bas son bosluk", "  Orkide  ", "orkide"},
		{"tire zaten var", "Mini-Buket", "mini-buket"},
		{"sadece rakam", "51", "51"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Slugify(tt.input))
		})
	}
}

func TestSlugify_TurkishCharsFully(t *testing.T) {
	assert.Equal(t, "cgiosu-cgiosu", Slugify("çğıöşü ÇĞİÖŞÜ"))
}

func TestSlugify_EmptyFallback(t *testing.T) {
	assert.Equal(t, "urun", Slugify(""))
	assert.Equal(t, "urun", Slugify("!!!"))
}
