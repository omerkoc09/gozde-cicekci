import type { CartItem } from '~/types/api'
import { addItem, cartLineKey, cartTotal, removeItem, setItemQuantity } from './cartLogic'

export { cartLineKey }

const STORAGE_KEY = 'cicekci_sepet'

/**
 * Sepet — localStorage'da yaşar, sunucuda carts tablosu YOK (spec §2.1).
 *
 * useState ile paylaşılan tek state: drawer, header rozeti ve sipariş
 * formu aynı sepeti görür.
 */
export function useCart() {
  const items = useState<CartItem[]>('sepet', () => [])

  // SSR'da localStorage yok; hydration sonrası okunur
  onMounted(() => {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw)
      return

    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed))
        items.value = parsed
    }
    catch {
      // Bozuk veri — sıfırla, patlatma
      localStorage.removeItem(STORAGE_KEY)
    }
  })

  function kaydet() {
    if (import.meta.client)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(items.value))
  }

  const count = computed(() => items.value.reduce((s, i) => s + i.quantity, 0))
  const itemsTotal = computed(() => cartTotal(items.value))

  function add(item: CartItem) {
    items.value = addItem(items.value, item)
    kaydet()
  }

  function remove(lineKey: string) {
    items.value = removeItem(items.value, lineKey)
    kaydet()
  }

  function setQuantity(lineKey: string, qty: number) {
    items.value = setItemQuantity(items.value, lineKey, qty)
    kaydet()
  }

  function clear() {
    items.value = []
    kaydet()
  }

  return { items, count, itemsTotal, add, remove, setQuantity, clear }
}
