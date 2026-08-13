/**
 * Giriş/kayıt sonrası dönülecek yolu ?donus parametresinden çözer.
 *
 * Checkout'tan "Giriş yapın" ile gelen kullanıcı, giriş sonrası sepetinin
 * başına (/siparis) dönmeli — varsayılan /hesabim'a düşerse sepetini
 * kaybettiğini sanar.
 *
 * GÜVENLİK: yalnızca tek "/" ile başlayan site içi yollar kabul edilir.
 * "//kotusite.com" (protokol-rölatif) ve "https://..." reddedilir; aksi
 * halde saldırgan kurbanı giriş sonrası kendi sitesine yönlendirebilirdi
 * (açık yönlendirme / open redirect).
 */
export function donusYolunuCoz(donus: unknown, varsayilan = '/hesabim'): string {
  return typeof donus === 'string' && /^\/(?!\/)/.test(donus) ? donus : varsayilan
}
