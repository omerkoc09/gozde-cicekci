/**
 * Türkiye cep telefonu doğrulama/normalizasyon — backend'deki
 * internal/customer/phone.go normalizePhone ile AYNI kuralı uygular.
 *
 * Buradaki kontrol yalnızca kullanıcıya anında geri bildirim içindir;
 * asıl güvenlik kontrolü backend'dedir (istemci doğrulaması atlatılabilir).
 * İki taraf ayrışırsa backend kazanır — bu dosyayı değiştirirken
 * phone.go'yu da güncelleyin.
 *
 * Kabul: 10 hane, 5 ile başlar. Ülke kodu (+90 / 90) ve baştaki 0 soyulur;
 * boşluk, tire, parantez, nokta gibi biçimlendirme karakterleri yok sayılır.
 */
export function telefonNormalize(raw: string): string | null {
  let d = ''
  for (const ch of (raw ?? '').trim()) {
    if (ch >= '0' && ch <= '9')
      d += ch
    else if (![' ', '-', '(', ')', '.', '/', '+'].includes(ch))
      return null // harf vb. — geçersiz
  }

  // Ülke kodu / baştaki sıfırı soy.
  if (d.length === 12 && d.startsWith('90'))
    d = d.slice(2)
  else if (d.length === 11 && d.startsWith('0'))
    d = d.slice(1)

  return d.length === 10 && d[0] === '5' ? d : null
}

/** Formda gösterilecek Türkçe hata mesajı (geçerliyse boş string). */
export function telefonHatasi(raw: string): string {
  if (!raw.trim())
    return 'Telefon gerekli.'
  return telefonNormalize(raw) === null
    ? 'Geçerli bir cep telefonu girin (örn. 0555 111 22 33).'
    : ''
}
