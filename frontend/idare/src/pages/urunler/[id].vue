<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import ProductImageManager from '@/components/ProductImageManager.vue'
import ProductOptionPicker from '@/components/ProductOptionPicker.vue'
import { useProducts } from '@/composables/useProducts'
import { useCategories } from '@/composables/useCategories'
import type {
  Product,
  ProductCreate,
  ProductOptionGroupLink,
  ProductUpdate,
  StokHareket,
} from '@/model/product'
import type { Axis, Category } from '@/model/category'
import { AXIS_LABELS } from '@/model/category'
import { ErrorPopup, SuccessToast } from '@/utils/Popup'
import { requiredValidator } from '@validators'

const route = useRoute('urunler-id')
const router = useRouter()

const productApi = useProducts()
const categoryApi = useCategories()

const rawId = computed(() => String(route.params.id))
const isNew = computed(() => rawId.value === 'yeni')
const productId = computed(() => (isNew.value ? 0 : Number(rawId.value)))

const loading = ref(false)
const saving = ref(false)
const formRef = ref<VForm>()

const product = ref<Product | null>(null)
const categories = ref<Category[]>([])

const form = ref({
  name: '',
  description: '',
  price: '',
  is_active: true,
  is_featured: false,
  category_ids: [] as number[],
  option_groups: [] as ProductOptionGroupLink[],

  track_stock: false,
  stock_quantity: 0,

  // Boş = indirim yok. Kaydederken ikisi de doluysa indirim açılır,
  // ikisi de boşsa (ve önceden indirim varsa) indirim kaldırılır.
  //
  // Tip null içeriyor: VTextField `clearable` ile temizlenince modele ''
  // değil null yazıyor. Okurken daima temizle() ile normalize edilir.
  discount_price: '' as string | null,
  discount_quota: '' as string | null,
})

// --- Stok hareketleri ---

const hareketler = ref<StokHareket[]>([])
const hareketlerYukleniyor = ref(false)

const SEBEP_ETIKET: Record<string, string> = {
  siparis: 'Sipariş',
  whatsapp_satisi: 'WhatsApp satışı',
  sayim_duzeltme: 'Sayım düzeltme',
  yeni_parti: 'Yeni parti',
  iptal_iade: 'İptal/İade',
  rezervasyon_iptal: 'Rezervasyon iptali',
}

const loadHareketler = async () => {
  if (isNew.value)
    return

  hareketlerYukleniyor.value = true

  const [err, data] = await productApi.movements(Number(rawId.value))

  hareketlerYukleniyor.value = false

  if (err)
    return ErrorPopup(err.message)

  hareketler.value = data ?? []
}

const tarihText = (iso: string) =>
  new Date(iso).toLocaleString('tr-TR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

/** İndirim aktif mi — kota dolduysa sönmüş sayılır (spec §3.3). */
const indirimAktif = computed(() => {
  const p = product.value
  if (!p || p.discount_price === null || p.discount_quota === null)
    return false

  return p.discount_sold < p.discount_quota
})

const kalanIndirimliAdet = computed(() => {
  const p = product.value
  if (!p || p.discount_quota === null)
    return 0

  return Math.max(p.discount_quota - p.discount_sold, 0)
})

/**
 * Alan değerini güvenle string'e indirger.
 *
 * VTextField `clearable` ile temizlendiğinde modele '' değil NULL yazıyor;
 * doğrudan .trim() çağırmak TypeError atıp save()'i sessizce yarıda kesiyordu
 * (indirim kaldırılamıyordu).
 */
const temizle = (v: string | null | undefined) => (v ?? '').trim()

// İndirimli fiyat normal fiyattan yüksekse esnaf muhtemelen yanlış giriyor.
const indirimFiyatUyarisi = computed(() => {
  const indirimli = Number.parseFloat(temizle(form.value.discount_price))
  const normal = Number.parseFloat(temizle(form.value.price))

  if (Number.isNaN(indirimli) || Number.isNaN(normal))
    return ''

  return indirimli >= normal ? 'İndirimli fiyat normal fiyattan düşük olmalı' : ''
})

const byAxis = (axis: Axis) =>
  categories.value
    .filter(c => c.axis === axis)
    .map(c => ({
      title: c.is_active ? c.name : `${c.name} (pasif)`,
      value: c.id,
    }))

const occasionOptions = computed(() => byAxis('occasion'))
const typeOptions = computed(() => byAxis('type'))

/** Seçili kategoriler tek listede tutuluyor; eksen bazlı VSelect'ler bunu böler. */
const selectedByAxis = (axis: Axis) => computed({
  get: () => form.value.category_ids.filter(id =>
    categories.value.find(c => c.id === id)?.axis === axis),
  set: (ids: number[]) => {
    const other = form.value.category_ids.filter(id =>
      categories.value.find(c => c.id === id)?.axis !== axis)

    form.value.category_ids = [...other, ...ids]
  },
})

const occasionSelected = selectedByAxis('occasion')
const typeSelected = selectedByAxis('type')

const priceValidator = (v: string) => {
  const n = Number.parseFloat(v)

  if (!v || Number.isNaN(n))
    return 'Fiyat girin'

  if (n < 0)
    return 'Fiyat negatif olamaz'

  return true
}

const loadCategories = async () => {
  const [err, data] = await categoryApi.list()
  if (err)
    return ErrorPopup(err.message)

  categories.value = data ?? []
}

const loadProduct = async () => {
  if (isNew.value)
    return

  loading.value = true

  const [err, data] = await productApi.get(productId.value)

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  product.value = data
  form.value = {
    name: data.name,
    description: data.description,
    price: data.price,
    is_active: data.is_active,
    is_featured: data.is_featured,
    category_ids: data.category_ids ?? [],
    option_groups: (data.option_groups ?? []).map(g => ({ group_id: g.id })),
    track_stock: data.track_stock,
    stock_quantity: data.stock_quantity,
    discount_price: data.discount_price ?? '',
    discount_quota: data.discount_quota === null ? '' : String(data.discount_quota),
  }
}

onMounted(async () => {
  await loadCategories()
  await loadProduct()
  await loadHareketler()
})

// Oluşturduktan sonra router.replace ile /urunler/yeni → /urunler/:id oluyor.
// Bileşen yeniden kurulmadığı için onMounted tekrar çalışmaz; ürünü burada
// yüklemezsek görsel bölümü kullanıcı sayfayı yenileyene kadar boş kalır.
watch(rawId, loadProduct)

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  // İndirimli fiyat normal fiyattan yüksekse kaydetme — sunucu bunu
  // reddetmiyor (teknik olarak geçerli), ama esnafın istediği bu değil.
  if (indirimFiyatUyarisi.value)
    return ErrorPopup(indirimFiyatUyarisi.value)

  const indirimliFiyat = temizle(form.value.discount_price)
  const indirimliAdet = temizle(form.value.discount_quota)

  if ((indirimliFiyat !== '') !== (indirimliAdet !== ''))
    return ErrorPopup('İndirimli fiyat ve adet birlikte girilmeli')

  // Fiyat API'ye string gidiyor: "1850.00" (float precision — spec §4.1).
  const payload: ProductCreate & ProductUpdate = {
    name: form.value.name,
    description: form.value.description,
    price: Number.parseFloat(form.value.price).toFixed(2),
    is_active: form.value.is_active,
    is_featured: form.value.is_featured,
    category_ids: form.value.category_ids,
    option_groups: form.value.option_groups,
  }

  // Stok ve indirim yalnızca mevcut üründe güncellenir — create ucu bu
  // alanları almıyor, ürün önce oluşturulup sonra düzenleniyor.
  if (!isNew.value) {
    payload.track_stock = form.value.track_stock
    payload.stock_quantity = form.value.track_stock ? Number(form.value.stock_quantity) : 0

    if (indirimliFiyat !== '' && indirimliAdet !== '') {
      payload.discount_price = Number.parseFloat(indirimliFiyat).toFixed(2)
      payload.discount_quota = Number(indirimliAdet)
    }
    else if (product.value?.discount_price) {
      // Alanlar boşaltıldı ve üründe indirim VARDI → indirimi kaldır.
      // product yüklenmemişse (undefined) hiçbir şey gönderilmez.
      payload.clear_discount = true
    }
  }

  saving.value = true

  if (isNew.value) {
    const [err, data] = await productApi.create(payload)

    saving.value = false

    if (err)
      return ErrorPopup(err.message)

    SuccessToast('Ürün oluşturuldu — artık görsel ekleyebilirsiniz')

    // Görsel yüklemek için ürünün var olması gerekiyor; düzenleme moduna geç.
    return router.replace({ name: 'urunler-id', params: { id: String(data.id) } })
  }

  const [err] = await productApi.update(productId.value, payload)

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Ürün güncellendi')
  await loadProduct()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <div class="d-flex align-center gap-2">
        <VBtn
          icon="tabler-arrow-left"
          variant="text"
          :to="{ name: 'root' }"
        />
        <h4 class="text-h4">
          {{ isNew ? 'Yeni Ürün' : form.name || 'Ürün' }}
        </h4>
      </div>

      <VBtn
        :loading="saving"
        prepend-icon="tabler-device-floppy"
        @click="save"
      >
        Kaydet
      </VBtn>
    </div>

    <VRow>
      <VCol
        cols="12"
        md="7"
      >
        <VCard :loading="loading">
          <VCardItem>
            <VCardTitle>Ürün Bilgileri</VCardTitle>
          </VCardItem>

          <VCardText>
            <VForm
              ref="formRef"
              @submit.prevent="save"
            >
              <VRow>
                <VCol cols="12">
                  <VTextField
                    v-model="form.name"
                    label="Ürün Adı"
                    :rules="[requiredValidator]"
                  />
                </VCol>

                <VCol
                  v-if="!isNew && product"
                  cols="12"
                >
                  <VTextField
                    :model-value="product.slug"
                    label="Link (slug)"
                    readonly
                    disabled
                    hint="Ürün adını değiştirirseniz yeni bir link oluşur, eski link yeni linke yönlendirilir."
                    persistent-hint
                  />
                </VCol>

                <VCol cols="12">
                  <VTextarea
                    v-model="form.description"
                    label="Açıklama"
                    rows="4"
                    auto-grow
                  />
                </VCol>

                <VCol
                  cols="12"
                  sm="6"
                >
                  <VTextField
                    v-model="form.price"
                    label="Fiyat (₺)"
                    type="number"
                    step="0.01"
                    min="0"
                    :rules="[priceValidator]"
                  />
                </VCol>

                <VCol
                  cols="12"
                  sm="6"
                  class="d-flex align-center gap-6"
                >
                  <VSwitch
                    v-model="form.is_active"
                    label="Aktif (sitede görünür)"
                    hide-details
                  />

                  <!--
                    Pasif ürün ana sayfada görünmez — öne çıkarmanın
                    anlamı kalmaz (kategorilerdeki kuralın aynısı).
                  -->
                  <VTooltip
                    :disabled="form.is_active"
                    text="Pasif ürün öne çıkarılamaz"
                    location="top"
                  >
                    <template #activator="{ props }">
                      <div v-bind="props">
                        <VSwitch
                          v-model="form.is_featured"
                          label="Ana sayfada öne çıkar"
                          :disabled="!form.is_active"
                          hide-details
                        />
                      </div>
                    </template>
                  </VTooltip>
                </VCol>

                <VCol cols="12">
                  <VSelect
                    v-model="occasionSelected"
                    :items="occasionOptions"
                    :label="AXIS_LABELS.occasion"
                    multiple
                    chips
                    closable-chips
                    no-data-text="Bu eksende kategori yok"
                  />
                </VCol>

                <VCol cols="12">
                  <VSelect
                    v-model="typeSelected"
                    :items="typeOptions"
                    :label="AXIS_LABELS.type"
                    multiple
                    chips
                    closable-chips
                    no-data-text="Bu eksende kategori yok"
                  />
                </VCol>

                <VCol cols="12">
                  <ProductOptionPicker v-model="form.option_groups" />
                </VCol>
              </VRow>
            </VForm>
          </VCardText>
        </VCard>

        <!--
          Stok ve indirim yalnızca kayıtlı üründe: create ucu bu
          alanları almıyor, ürün önce oluşturuluyor.
        -->
        <VCard
          v-if="!isNew"
          class="mt-6"
        >
          <VCardItem>
            <VCardTitle>Stok</VCardTitle>
          </VCardItem>

          <VCardText>
            <VSwitch
              v-model="form.track_stock"
              label="Stok takibi yap"
              hide-details
              density="comfortable"
            />
            <div class="text-caption text-medium-emphasis mt-1">
              Kapalıysa ürün sınırsız satılır, "tükendi" durumuna hiç düşmez.
            </div>

            <VRow
              v-if="form.track_stock"
              class="mt-2"
            >
              <VCol
                cols="12"
                sm="6"
              >
                <VTextField
                  v-model.number="form.stock_quantity"
                  type="number"
                  min="0"
                  label="Stok adedi"
                />
              </VCol>
              <VCol
                cols="12"
                sm="6"
                class="d-flex align-center"
              >
                <div
                  v-if="(product?.stock_reserved ?? 0) > 0"
                  class="text-body-2 text-medium-emphasis"
                >
                  <VIcon
                    icon="tabler-clock"
                    size="16"
                    class="me-1"
                  />
                  {{ product?.stock_reserved }} adet ödeme bekliyor
                </div>
              </VCol>
            </VRow>
          </VCardText>
        </VCard>

        <VCard
          v-if="!isNew"
          class="mt-6"
        >
          <VCardItem>
            <VCardTitle>İndirim</VCardTitle>
          </VCardItem>

          <VCardText>
            <div class="text-body-2 text-medium-emphasis mb-3">
              İndirimli ürünler sitede ayrı bir sayfada listelenir. Belirtilen
              adet satıldığında indirim kendiliğinden kalkar.
            </div>

            <VRow>
              <VCol
                cols="12"
                sm="6"
              >
                <VTextField
                  v-model="form.discount_price"
                  type="number"
                  min="0"
                  step="0.01"
                  label="İndirimli fiyat"
                  suffix="TL"
                  :error-messages="indirimFiyatUyarisi"
                  clearable
                />
              </VCol>
              <VCol
                cols="12"
                sm="6"
              >
                <VTextField
                  v-model="form.discount_quota"
                  type="number"
                  min="1"
                  label="Kaç adet indirimli"
                  clearable
                />
              </VCol>
            </VRow>

            <div
              v-if="product?.discount_price"
              class="d-flex align-center ga-3 mt-2"
            >
              <VChip
                :color="indirimAktif ? 'success' : 'default'"
                size="small"
                label
              >
                {{ indirimAktif ? 'İndirim aktif' : 'Kota doldu — indirim kalktı' }}
              </VChip>
              <span class="text-body-2 text-medium-emphasis">
                Satılan: {{ product.discount_sold }} · Kalan: {{ kalanIndirimliAdet }}
              </span>
            </div>

            <div class="text-caption text-medium-emphasis mt-3">
              İndirimi kaldırmak için iki alanı da boşaltıp kaydedin.
            </div>
          </VCardText>
        </VCard>
      </VCol>

      <VCol
        cols="12"
        md="5"
      >
        <VCard>
          <VCardItem>
            <VCardTitle>Görseller</VCardTitle>
          </VCardItem>

          <VCardText>
            <!--
              Görsel yüklemek için ürünün veritabanında olması gerekiyor;
              backend ürün yoksa 404 dönüyor.
            -->
            <VAlert
              v-if="isNew"
              type="info"
              variant="tonal"
              density="compact"
            >
              Görsel eklemek için önce ürünü kaydedin.
            </VAlert>

            <ProductImageManager
              v-else-if="product"
              :product-id="product.id"
              :images="product.images ?? []"
              @update="loadProduct"
            />
          </VCardText>
        </VCard>

        <VCard
          v-if="!isNew"
          class="mt-6"
        >
          <VCardItem>
            <VCardTitle>Stok Hareketleri</VCardTitle>
          </VCardItem>

          <VCardText>
            <VProgressLinear
              v-if="hareketlerYukleniyor"
              indeterminate
            />

            <div
              v-else-if="!hareketler.length"
              class="text-body-2 text-medium-emphasis"
            >
              Henüz stok hareketi yok.
            </div>

            <VTable
              v-else
              density="compact"
            >
              <thead>
                <tr>
                  <th>Tarih</th>
                  <th class="text-end">
                    Değişim
                  </th>
                  <th>Sebep</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="h in hareketler"
                  :key="h.id"
                >
                  <td class="text-caption text-no-wrap">
                    {{ tarihText(h.created_at) }}
                  </td>
                  <td class="text-end">
                    <!-- delta 0: rezervasyon iptali — fiziksel stok değişmedi -->
                    <span
                      v-if="h.delta === 0"
                      class="text-medium-emphasis"
                    >—</span>
                    <span
                      v-else
                      :class="h.delta > 0 ? 'text-success' : 'text-error'"
                      class="font-weight-medium"
                    >
                      {{ h.delta > 0 ? '+' : '' }}{{ h.delta }}
                    </span>
                  </td>
                  <td class="text-caption">
                    {{ SEBEP_ETIKET[h.reason] ?? h.reason }}
                    <VChip
                      v-if="h.was_discounted"
                      size="x-small"
                      color="error"
                      label
                      class="ms-1"
                    >
                      indirimli
                    </VChip>
                    <div
                      v-if="h.note"
                      class="text-medium-emphasis"
                    >
                      {{ h.note }}
                    </div>
                  </td>
                </tr>
              </tbody>
            </VTable>
          </VCardText>
        </VCard>
      </VCol>
    </VRow>
  </div>
</template>
