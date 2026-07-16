import annelerGunu from '~/assets/img/kategori-anneler-gunu.webp'
import sevgiliye from '~/assets/img/kategori-sevgiliye.webp'
import dugunAcilis from '~/assets/img/kategori-dugun-acilis.webp'
import sozKizIsteme from '~/assets/img/kategori-soz-kiz-isteme.webp'
import type { Category } from '~/types/api'
import { kategoriGorselIndex } from '~/utils/kategoriGorsel'

/**
 * Kategori görselini seçer: panelden yüklenmiş görsel varsa O kullanılır,
 * yoksa aşağıdaki yedek havuza düşülür.
 *
 * Yedek havuz yalnızca 4 ÖZEL GÜN fotoğrafı içeriyor; çiçek türleri
 * (Orkideler, Buketler...) için uygun görsel yok — o kategorilerde
 * dönüşümlü atama alakasız bir fotoğraf verir. Doğrusu panelden görsel
 * yüklemek; bu havuz yalnızca kart boş kalmasın diye duruyor.
 *
 * Görseller referans mockup'lardan YEREL asset'e çevrildi — referanstaki gibi
 * Google CDN'den hotlink edilmiyor (o URL'ler kalıcı değil, spec §4.2).
 */
const HAVUZ = [annelerGunu, sevgiliye, dugunAcilis, sozKizIsteme]

/** Yedek havuzdan görsel seçer — panelde görsel yoksa kullanılır. */
export function kategoriYedekGorseli(slug: string, name: string, index = 0): string {
  return HAVUZ[kategoriGorselIndex(slug, name, index, HAVUZ.length)]!
}

export function kategoriGorseli(kategori: Category, index = 0): string {
  return kategori.url_900 || kategoriYedekGorseli(kategori.slug, kategori.name, index)
}
