import annelerGunu from '~/assets/img/kategori-anneler-gunu.webp'
import sevgiliye from '~/assets/img/kategori-sevgiliye.webp'
import dugunAcilis from '~/assets/img/kategori-dugun-acilis.webp'
import sozKizIsteme from '~/assets/img/kategori-soz-kiz-isteme.webp'
import { kategoriGorselIndex } from '~/utils/kategoriGorsel'

/**
 * Kategori görsellerini asset'lere bağlar. Seçim mantığı utils'te (saf TS,
 * test edilebilir); burası yalnızca havuzu tutuyor.
 *
 * Görseller referans mockup'lardan YEREL asset'e çevrildi — referanstaki gibi
 * Google CDN'den hotlink edilmiyor (o URL'ler kalıcı değil, spec §4.2).
 * Gözde Tasarım'ın gerçek fotoğrafları gelince yalnızca bu import'lar değişir.
 */
const HAVUZ = [annelerGunu, sevgiliye, dugunAcilis, sozKizIsteme]

export function kategoriGorseli(slug: string, name: string, index = 0): string {
  return HAVUZ[kategoriGorselIndex(slug, name, index, HAVUZ.length)]!
}
