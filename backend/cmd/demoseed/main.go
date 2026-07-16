// Command demoseed geliştirme veritabanına örnek kategori ve ürün basar.
//
// Amaç: tasarımı ve akışı dolu bir vitrinle görebilmek. Boş veritabanında
// ana sayfa, koleksiyon ve kategori sayfaları boş durumlarını gösterir —
// gerçek görünümü değerlendirmek mümkün olmaz.
//
// Bu komut cmd/seed'den AYRI: seed prod'da admin kullanıcı oluşturur ve
// oraya demo veri mantığı karışmamalı.
//
// GÜVENLİK: APP_ENV=production ise çalışmayı reddeder. Görseller
// uploads/products/<key>/{400,1200}.jpg altında hazır bulunmalıdır.
//
// Kullanım:
//
//	go run ./cmd/demoseed          # ekle (mevcut demo veriyi temizler)
//	go run ./cmd/demoseed -clean   # yalnızca temizle
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/omerkoc/cicekci/pkg/config"
	"github.com/omerkoc/cicekci/pkg/database"
)

type demoKategori struct {
	Name     string
	Slug     string
	Axis     string
	Featured bool
	Sort     int
}

type demoUrun struct {
	Name     string
	Slug     string
	Desc     string
	Price    string
	Kategori []string // kategori slug'ları
	Images   []string // image_key'ler; ilki kapak
}

// Görsel key'leri: uploads/products/<key>/{400,1200}.jpg olarak önceden
// yerleştirildi. Referans mockup'lardan alınan fotoğraflar (spec §4.2) —
// Gözde Tasarım'ın gerçek ürünleri değil, geçici.
const (
	imgGul15     = "53c5415ffdfb7789"
	imgGul25     = "4b3c4a73ebfa0a51"
	imgOrkideB   = "772ca8807807c512"
	imgGulBuket  = "ff7193ae475b1c09"
	imgGulYakin  = "c4a2cfcd3aa99144"
	imgGulKurdel = "f98182e42e48506e"
	imgGulKutu   = "5dcd0573bf401ba3"
	imgPastel    = "a31993fc184302f2"
	imgKarma     = "7beda764e0e5c2ce"
)

var kategoriler = []demoKategori{
	// occasion — ana sayfada "Gönderim Türüne Göre" bölümünde çıkanlar
	{"Anneler Günü", "anneler-gunu", "occasion", true, 1},
	{"Sevgiliye Çiçek", "sevgiliye-cicek", "occasion", true, 2},
	{"Düğün / Açılış", "dugun-acilis", "occasion", true, 3},
	{"Söz / Kız İsteme", "soz-kiz-isteme", "occasion", true, 4},
	{"Doğum Günü", "dogum-gunu", "occasion", false, 5},
	{"Geçmiş Olsun", "gecmis-olsun", "occasion", false, 6},

	// type
	{"Buketler", "buketler", "type", false, 1},
	{"Orkideler", "orkideler", "type", false, 2},
	{"Tasarım Çiçekler", "tasarim-cicekler", "type", false, 3},
	{"Kutuda Çiçek", "kutuda-cicek", "type", false, 4},
}

var urunler = []demoUrun{
	{
		Name:  "15'li Kırmızı Gül Buketi",
		Slug:  "15li-kirmizi-gul-buketi",
		Desc:  "Aşkın ve tutkunun en saf hali. Özenle seçilmiş 15 adet birinci sınıf Ekvador cinsi kırmızı gül, özel tasarım mat ithal kağıt ve lüks kurdele detaylarıyla hazırlanmıştır. Sevdiklerinize unutulmaz bir an yaşatmak için zarif ve zamansız bir tercih.",
		Price: "2749.00",
		Kategori: []string{"sevgiliye-cicek", "buketler"},
		Images:   []string{imgGul15, imgGulYakin, imgGulKurdel},
	},
	{
		Name:  "25'li Gül Aranjmanı",
		Slug:  "25li-gul-aranjmani",
		Desc:  "Yirmi beş adet premium kırmızı gülün, zarif gri-gümüş ambalaj içinde sunulduğu gösterişli bir aranjman. Sıkı sarımı ve kusursuz tazeliğiyle en özel anlarınıza yakışır.",
		Price: "3749.00",
		Kategori: []string{"sevgiliye-cicek", "soz-kiz-isteme", "buketler"},
		Images:   []string{imgGul25, imgGulBuket},
	},
	{
		Name:  "Beyaz Orkide Tasarım",
		Slug:  "beyaz-orkide-tasarim",
		Desc:  "Çift dallı beyaz Phalaenopsis orkide, dokulu koyu seramik saksıda. Geniş yeşil yaprakları ve doğal ahşap detaylarıyla klasik ve zamansız bir hediye sunumu.",
		Price: "1849.00",
		Kategori: []string{"dogum-gunu", "orkideler"},
		Images:   []string{imgOrkideB},
	},
	{
		Name:  "Soft Pastel Harmony",
		Slug:  "soft-pastel-harmony",
		Desc:  "Pudra pembesi güller, şakayıklar ve mevsim yeşillikleriyle hazırlanan yumuşak tonlu bir buket. Anneler Günü ve doğum günleri için zarif bir seçim.",
		Price: "2450.00",
		Kategori: []string{"anneler-gunu", "buketler"},
		Images:   []string{imgPastel, imgGulKutu},
	},
	{
		Name:  "Kırmızı & Beyaz Karma Buket",
		Slug:  "kirmizi-beyaz-karma-buket",
		Desc:  "Kırmızı güller, beyaz laleler ve bordo orkidelerin bir arada sunulduğu canlı ve dengeli bir aranjman. Atölyemizde el işçiliğiyle hazırlanır.",
		Price: "2999.00",
		Kategori: []string{"sevgiliye-cicek", "tasarim-cicekler"},
		Images:   []string{imgKarma, imgGulBuket},
	},
	{
		Name:  "Kutuda Kırmızı Gül",
		Slug:  "kutuda-kirmizi-gul",
		Desc:  "Silindir kutu içinde, yan yana dizilmiş kırmızı güller. Uzun ömürlü sunumu ve şık kutusuyla ofis ve ev teslimatları için ideal.",
		Price: "2250.00",
		Kategori: []string{"dogum-gunu", "kutuda-cicek"},
		Images:   []string{imgGulKutu, imgGulYakin},
	},
	{
		Name:  "Tek Dal Gül Zarafeti",
		Slug:  "tek-dal-gul-zarafeti",
		Desc:  "Üzerinde çiy damlalarıyla tek bir kusursuz kırmızı gül. Sade, doğrudan ve etkileyici — küçük jestlerin en zarif hali.",
		Price: "649.00",
		Kategori: []string{"sevgiliye-cicek", "tasarim-cicekler"},
		Images:   []string{imgGulYakin},
	},
	{
		Name:  "Pembe Kurdeleli Buket",
		Slug:  "pembe-kurdeleli-buket",
		Desc:  "Krem ambalaj ve pudra pembesi saten kurdeleyle sarılmış zarif bir buket. Geçmiş olsun ve teşekkür gönderimleri için sıcak bir seçim.",
		Price: "1950.00",
		Kategori: []string{"gecmis-olsun", "anneler-gunu", "buketler"},
		Images:   []string{imgGulKurdel, imgPastel},
	},
}

func main() {
	clean := flag.Bool("clean", false, "yalnızca demo veriyi sil")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Demo veri prod'a asla basılmamalı.
	if cfg.AppEnv == "production" {
		log.Fatal("demoseed production ortamında çalıştırılamaz")
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanı: %v", err)
	}
	defer pool.Close()

	// Tek transaction: yarım kalmış demo veri bırakmaz.
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := temizle(ctx, tx); err != nil {
		log.Fatalf("temizle: %v", err)
	}

	if *clean {
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit: %v", err)
		}
		fmt.Println("Demo veri silindi.")
		return
	}

	if err := gorselleriDogrula(cfg.UploadDir); err != nil {
		log.Fatalf("görsel: %v", err)
	}

	katID, err := kategorileriEkle(ctx, tx)
	if err != nil {
		log.Fatalf("kategori: %v", err)
	}

	n, err := urunleriEkle(ctx, tx, katID)
	if err != nil {
		log.Fatalf("ürün: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("Demo veri hazır: %d kategori, %d ürün.\n", len(kategoriler), n)
}

// temizle demo veriyi siler. product_slugs / product_images /
// product_categories FK'ları ON DELETE CASCADE — ürünü silmek yeter.
func temizle(ctx context.Context, tx pgx.Tx) error {
	for _, u := range urunler {
		if _, err := tx.Exec(ctx,
			`DELETE FROM products WHERE id IN (
			   SELECT product_id FROM product_slugs WHERE slug = $1)`, u.Slug); err != nil {
			return err
		}
	}
	for _, k := range kategoriler {
		if _, err := tx.Exec(ctx, `DELETE FROM categories WHERE slug = $1`, k.Slug); err != nil {
			return err
		}
	}
	return nil
}

// gorselleriDogrula her key için iki boyutun da diskte olduğunu kontrol eder.
// Eksikse ürünler kırık görselle görünürdü — sessizce geçmek yerine hata ver.
func gorselleriDogrula(uploadDir string) error {
	seen := map[string]bool{}
	for _, u := range urunler {
		for _, k := range u.Images {
			if seen[k] {
				continue
			}
			seen[k] = true
			for _, size := range []string{"400", "1200"} {
				p := filepath.Join(uploadDir, "products", k, size+".jpg")
				if _, err := os.Stat(p); err != nil {
					return fmt.Errorf("%s bulunamadı", p)
				}
			}
		}
	}
	return nil
}

func kategorileriEkle(ctx context.Context, tx pgx.Tx) (map[string]int64, error) {
	ids := make(map[string]int64, len(kategoriler))
	for _, k := range kategoriler {
		var id int64
		err := tx.QueryRow(ctx,
			`INSERT INTO categories (name, slug, axis, is_active, is_featured, sort_order)
			 VALUES ($1, $2, $3, true, $4, $5) RETURNING id`,
			k.Name, k.Slug, k.Axis, k.Featured, k.Sort).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k.Slug, err)
		}
		ids[k.Slug] = id
	}
	return ids, nil
}

func urunleriEkle(ctx context.Context, tx pgx.Tx, katID map[string]int64) (int, error) {
	for _, u := range urunler {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO products (name, description, price, is_active)
			 VALUES ($1, $2, $3, true) RETURNING id`,
			u.Name, u.Desc, u.Price).Scan(&id); err != nil {
			return 0, fmt.Errorf("%s: %w", u.Slug, err)
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
			u.Slug, id); err != nil {
			return 0, fmt.Errorf("%s slug: %w", u.Slug, err)
		}

		for _, ks := range u.Kategori {
			cid, ok := katID[ks]
			if !ok {
				return 0, fmt.Errorf("%s: bilinmeyen kategori %q", u.Slug, ks)
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2)`,
				id, cid); err != nil {
				return 0, fmt.Errorf("%s kategori: %w", u.Slug, err)
			}
		}

		// sort_order 0 = kapak (spec §4.4)
		for i, key := range u.Images {
			if _, err := tx.Exec(ctx,
				`INSERT INTO product_images (product_id, image_key, sort_order) VALUES ($1, $2, $3)`,
				id, key, i); err != nil {
				return 0, fmt.Errorf("%s görsel: %w", u.Slug, err)
			}
		}
	}
	return len(urunler), nil
}
