/**
 * Hesap sayfalarının statik verisi (spec §2.1).
 *
 * ⚠️ Bu veri SAHTE. Backend'de users/orders/addresses/favorites yok ve MVP
 * spec'i (§3.3) üyeliği gerekçeli olarak reddediyor. Hesap sayfaları Faz 2
 * önizlemesi / demo amacıyla, kullanıcı kararıyla inert olarak yapıldı.
 *
 * Tek yerde tutuluyor ki Faz 2'de backend gelince buradan gerçek composable'a
 * geçiş tek dosyayı ilgilendirsin; ekranlar değişmesin.
 *
 * İsim referans mockup'lardan alındı (Elif Yılmaz).
 */

export interface MockAdres {
  id: number
  tur: 'fatura' | 'teslimat'
  ad: string
  satir: string
  ilce: string
  postaKodu: string
  telefon: string
}

export const MOCK_KULLANICI = {
  ad: 'Elif',
  tamAd: 'Elif Yılmaz',
  eposta: 'elif.yilmaz@example.com',
  telefon: '+90 555 123 4567',
} as const

export const MOCK_ADRESLER: MockAdres[] = [
  {
    id: 1,
    tur: 'fatura',
    ad: 'Elif Yılmaz',
    satir: 'Gül Sokak No: 15, D: 4',
    ilce: 'Kadıköy, İstanbul',
    postaKodu: '34710',
    telefon: '+90 555 123 4567',
  },
  {
    id: 2,
    tur: 'teslimat',
    ad: 'Elif Yılmaz',
    satir: 'Gül Sokak No: 15, D: 4',
    ilce: 'Kadıköy, İstanbul',
    postaKodu: '34710',
    telefon: '+90 555 123 4567',
  },
]
