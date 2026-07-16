/**
 * API fiyatı string döner ("1850.00") — float precision sorunu olmasın diye.
 * Burada sadece GÖSTERİM formatlanıyor; hesaplama yapılmıyor.
 */
export function formatPrice(price: string): string {
  const n = Number(price)
  if (!price || Number.isNaN(n))
    return ''

  const hasKurus = n % 1 !== 0

  return `${n.toLocaleString('tr-TR', {
    minimumFractionDigits: hasKurus ? 2 : 0,
    maximumFractionDigits: 2,
  })} ₺`
}

/** WhatsApp mesajında kullanılan format — şimdilik aynı. */
export function whatsappPrice(price: string): string {
  return formatPrice(price)
}
