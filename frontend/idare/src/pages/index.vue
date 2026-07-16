<script setup lang="ts">
import { useProducts } from '@/composables/useProducts'
import { useCategories } from '@/composables/useCategories'
import type { Product } from '@/model/product'
import type { Category } from '@/model/category'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'

const productApi = useProducts()
const categoryApi = useCategories()
const router = useRouter()

const loading = ref(false)
const products = ref<Product[]>([])
const categories = ref<Category[]>([])

const categoryNames = computed(() => {
  const map = new Map<number, string>()

  categories.value.forEach(c => map.set(c.id, c.name))

  return map
})

const headers = [
  { title: '', key: 'cover', sortable: false, width: 72 },
  { title: 'Ürün', key: 'name' },
  { title: 'Fiyat', key: 'price', width: 130 },
  { title: 'Kategoriler', key: 'category_ids', sortable: false },
  { title: 'Durum', key: 'is_active', width: 110 },
  { title: 'Vitrin', key: 'is_featured', sortable: false, width: 110 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 110 },
]

/** Kapak = sort_order en küçük görsel (spec §4.4). */
const coverOf = (p: Product) =>
  [...(p.images ?? [])].sort((a, b) => a.sort_order - b.sort_order)[0]

const priceText = (price: string) => {
  const n = Number.parseFloat(price)

  return Number.isNaN(n)
    ? price
    : `${n.toLocaleString('tr-TR', { minimumFractionDigits: 2 })} ₺`
}

const goToProduct = (id: number | string) =>
  router.push({ name: 'urunler-id', params: { id: String(id) } })

const load = async () => {
  loading.value = true

  // Esnafın 40-100 ürünü olacak — tek sayfada gösteriliyor, sayfalama yok.
  // Backend toplam sayı dönmüyor; ürün sayısı büyürse Faz 2'de eklenir.
  const [[pErr, pData], [cErr, cData]] = await Promise.all([
    productApi.list(1, 100),
    categoryApi.list(),
  ])

  loading.value = false

  if (pErr)
    return ErrorPopup(pErr.message)

  if (cErr)
    return ErrorPopup(cErr.message)

  products.value = pData ?? []
  categories.value = cData ?? []
}

/**
 * Vitrin switch'i anında kaydeder — ürünü düzenlemeye girmeye gerek yok.
 * İyimser değil: yanıt gelene kadar satırdaki değer eski kalır.
 */
const toggleFeatured = async (p: Product, value: boolean) => {
  const [err] = await productApi.update(p.id, { is_featured: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

onMounted(load)

const remove = async (p: Product) => {
  const ok = await ConfirmPopup(
    `"${p.name}" ve tüm görselleri silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [err] = await productApi.remove(p.id)
  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Ürün silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Ürünler
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="goToProduct('yeni')"
      >
        Yeni Ürün
      </VBtn>
    </div>

    <VCard>
      <VDataTable
        :headers="headers"
        :items="products"
        :loading="loading"
        :items-per-page="-1"
        no-data-text="Henüz ürün yok"
        loading-text="Yükleniyor..."
      >
        <template #item.cover="{ item }">
          <VAvatar
            size="48"
            rounded="lg"
            :color="coverOf(item) ? undefined : 'secondary'"
            variant="tonal"
            :class="{ 'opacity-50': !item.is_active }"
          >
            <VImg
              v-if="coverOf(item)"
              :src="coverOf(item).url_400"
              cover
            />
            <VIcon
              v-else
              icon="tabler-photo-off"
              size="20"
            />
          </VAvatar>
        </template>

        <template #item.name="{ item }">
          <span
            class="font-weight-medium"
            :class="{ 'text-disabled': !item.is_active }"
          >
            {{ item.name }}
          </span>
        </template>

        <template #item.price="{ item }">
          <span :class="{ 'text-disabled': !item.is_active }">
            {{ priceText(item.price) }}
          </span>
        </template>

        <template #item.category_ids="{ item }">
          <span
            v-if="!item.category_ids?.length"
            class="text-disabled"
          >—</span>

          <template v-else>
            <VChip
              v-for="id in item.category_ids"
              :key="id"
              size="x-small"
              class="me-1 my-1"
            >
              {{ categoryNames.get(id) ?? `#${id}` }}
            </VChip>
          </template>
        </template>

        <template #item.is_active="{ item }">
          <VChip
            :color="item.is_active ? 'success' : undefined"
            size="small"
          >
            {{ item.is_active ? 'Aktif' : 'Pasif' }}
          </VChip>
        </template>

        <template #item.is_featured="{ item }">
          <!-- Pasif ürün ana sayfada görünmez — öne çıkarmanın anlamı yok. -->
          <VTooltip
            :disabled="item.is_active"
            text="Pasif ürün öne çıkarılamaz"
            location="top"
          >
            <template #activator="{ props }">
              <div
                v-bind="props"
                class="d-inline-block"
              >
                <VSwitch
                  :model-value="item.is_featured"
                  :disabled="!item.is_active"
                  density="compact"
                  hide-details
                  @update:model-value="toggleFeatured(item, $event as boolean)"
                />
              </div>
            </template>
          </VTooltip>
        </template>

        <template #item.actions="{ item }">
          <VBtn
            icon="tabler-pencil"
            variant="text"
            size="small"
            @click="goToProduct(item.id)"
          />
          <VBtn
            icon="tabler-trash"
            variant="text"
            size="small"
            color="error"
            @click="remove(item)"
          />
        </template>
      </VDataTable>
    </VCard>
  </div>
</template>
