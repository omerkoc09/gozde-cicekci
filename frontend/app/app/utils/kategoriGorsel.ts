/**
 * Kategori kartı görseli seçimi (spec §4.2).
 *
 * Kategoriler admin panelden yönetiliyor; slug'ları önceden bilinmiyor
 * (seed'de kategori yok, esnaf kendi oluşturuyor). Bu yüzden slug'a sabit
 * eşleme kırılgan olurdu — kategori adı değişince görsel düşerdi.
 *
 * İki katmanlı çözüm: önce ada/slug'a göre anahtar kelime araması, tutmazsa
 * sıraya göre dönüşümlü atama. Hangi kategori gelirse gelsin kart görselsiz
 * kalmaz.
 *
 * Asset import'ları BİLEREK burada değil (bkz. kategoriGorsel.client.ts):
 * bu dosya saf TS kalsın ki mantık Vite'a ihtiyaç duymadan test edilebilsin.
 *
 * Faz 2: `categories.image_url` kolonu eklenince tüm bu dosya kalkar.
 */

/** Anahtar kelime → havuzdaki görselin sırası */
const ANAHTARLAR: Array<[RegExp, number]> = [
  [/anne/i, 0],
  [/sevgili|aşk|ask|romantik|yıldönüm|yildonum/i, 1],
  [/düğün|dugun|açılış|acilis|nikah|organizasyon/i, 2],
  [/söz|soz|nişan|nisan|kız iste|kiz iste/i, 3],
]

/**
 * Kategoriye hangi görselin düşeceğini seçer.
 *
 * @param slug kategori slug'ı
 * @param name kategori adı — anahtar kelime araması için
 * @param index listedeki sırası — hiçbir anahtar tutmazsa dönüşümlü atama
 * @param havuzBoyu kullanılabilir görsel sayısı
 * @returns havuzdaki görselin index'i
 */
export function kategoriGorselIndex(
  slug: string,
  name: string,
  index = 0,
  havuzBoyu = 4,
): number {
  const eslesme = ANAHTARLAR.find(([re]) => re.test(name) || re.test(slug))

  if (eslesme && eslesme[1] < havuzBoyu)
    return eslesme[1]

  return index % havuzBoyu
}
