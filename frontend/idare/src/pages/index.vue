<script setup lang="ts">
import { useProducts } from '@/composables/useProducts'
import { useCategories } from '@/composables/useCategories'
import type { Product } from '@/model/product'
import type { Category } from '@/model/category'
import { formatTutar as priceText } from '@/utils/Currency'
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

// Satır başına en fazla bu kadar kategori çipi gösterilir — fazlası "+N"
// olarak özetlenir, yoksa çok kategorili ürünlerde satır şişer.
const MAX_KATEGORI_CIP = 2

// Filtreler client-side: liste zaten tek seferde çekiliyor (limit 100),
// esnafın 40-100 ürünü backend'e ek istek atmadan filtrelenebilir.
const arama = ref('')
const kategoriFiltresi = ref<number | null>(null)
const durumFiltresi = ref<'' | 'active' | 'passive'>('')

const kategoriSecenekleri = computed(() =>
  categories.value.map(c => ({ title: c.name, value: c.id })))

const filtrelenmisUrunler = computed(() => products.value.filter(p => {
  const kategoriUyuyor = kategoriFiltresi.value === null || !!p.category_ids?.includes(kategoriFiltresi.value)
  const durumUyuyor = durumFiltresi.value === '' || (durumFiltresi.value === 'active') === p.is_active

  return kategoriUyuyor && durumUyuyor
}))

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
      <VCardText>
        <VRow>
          <VCol
            cols="12"
            md="5"
          >
            <VTextField
              v-model="arama"
              prepend-inner-icon="tabler-search"
              placeholder="Ürün ara..."
              hide-details
              clearable
            />
          </VCol>
          <VCol
            cols="12"
            sm="6"
            md="4"
          >
            <VSelect
              v-model="kategoriFiltresi"
              :items="kategoriSecenekleri"
              placeholder="Kategori"
              hide-details
              clearable
            />
          </VCol>
          <VCol
            cols="12"
            sm="6"
            md="3"
          >
            <VBtnToggle
              v-model="durumFiltresi"
              density="compact"
              divided
            >
              <VBtn value="">
                Hepsi
              </VBtn>
              <VBtn value="active">
                Aktif
              </VBtn>
              <VBtn value="passive">
                Pasif
              </VBtn>
            </VBtnToggle>
          </VCol>
        </VRow>
      </VCardText>

      <VDataTable
        :headers="headers"
        :items="filtrelenmisUrunler"
        :search="arama"
        :loading="loading"
        :items-per-page="-1"
        no-data-text="Bu filtreye uyan ürün yok"
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
              v-for="id in item.category_ids.slice(0, MAX_KATEGORI_CIP)"
              :key="id"
              size="x-small"
              class="me-1 my-1"
            >
              {{ categoryNames.get(id) ?? `#${id}` }}
            </VChip>

            <VTooltip
              v-if="item.category_ids.length > MAX_KATEGORI_CIP"
              :text="item.category_ids
                .slice(MAX_KATEGORI_CIP)
                .map(id => categoryNames.get(id) ?? `#${id}`)
                .join(', ')"
              location="top"
            >
              <template #activator="{ props }">
                <VChip
                  v-bind="props"
                  size="x-small"
                  variant="tonal"
                  class="my-1"
                >
                  +{{ item.category_ids.length - MAX_KATEGORI_CIP }}
                </VChip>
              </template>
            </VTooltip>
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
