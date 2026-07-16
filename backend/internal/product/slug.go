package product

import (
	"regexp"
	"strings"
)

var turkishReplacer = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i",
	"İ", "i", "i", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multipleDashes  = regexp.MustCompile(`-{2,}`)
)

// Slugify bir ürün adını URL slug'ına çevirir.
// Türkçe karakterler ASCII karşılıklarına dönüşür.
// Sonuç boş kalırsa "urun" döner.
func Slugify(name string) string {
	s := turkishReplacer.Replace(name)
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = multipleDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if s == "" {
		return "urun"
	}
	return s
}
