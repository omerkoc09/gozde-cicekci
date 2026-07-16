import type { NavGroup } from '@layouts/types'

/**
 * Yetkilendirme yok — tek admin var (spec §4.5).
 *
 * Şablon @casl ile üç rollü (admin/teacher/parent) bir yetki sistemi
 * kuruyordu; okul projesinden kalmıştı. @layouts'taki beş navigasyon
 * bileşeni bu dosyadan `can`/`canViewNavMenuGroup` alıyor, o yüzden
 * dosya duruyor ama her şeye izin veriyor. Menü görünürlüğü artık
 * yalnızca navigation/vertical/index.ts'in içeriğine bağlı.
 *
 * Bileşenler `can(item.action, item.subject)` diye çağırıyor; imza
 * korunuyor ama argümanlara bakılmıyor.
 */
export const can = (..._: unknown[]) => true

/** Grup, görünür çocuğu varsa görünür. */
export const canViewNavMenuGroup = (item: NavGroup) =>
  item.children.some(i => can(i.action, i.subject))

/** Her rota gezilebilir; oturum kontrolü router guard'ında (1.router/index.ts). */
export const canNavigate = (..._: unknown[]) => true
