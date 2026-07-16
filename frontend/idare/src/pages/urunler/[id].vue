<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import ProductImageManager from '@/components/ProductImageManager.vue'
import { useProducts } from '@/composables/useProducts'
import { useCategories } from '@/composables/useCategories'
import type { Product } from '@/model/product'
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
  category_ids: [] as number[],
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
    category_ids: data.category_ids ?? [],
  }
}

onMounted(async () => {
  await loadCategories()
  await loadProduct()
})

// Oluşturduktan sonra router.replace ile /urunler/yeni → /urunler/:id oluyor.
// Bileşen yeniden kurulmadığı için onMounted tekrar çalışmaz; ürünü burada
// yüklemezsek görsel bölümü kullanıcı sayfayı yenileyene kadar boş kalır.
watch(rawId, loadProduct)

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  // Fiyat API'ye string gidiyor: "1850.00" (float precision — spec §4.1).
  const payload = {
    name: form.value.name,
    description: form.value.description,
    price: Number.parseFloat(form.value.price).toFixed(2),
    is_active: form.value.is_active,
    category_ids: form.value.category_ids,
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
                  class="d-flex align-center"
                >
                  <VSwitch
                    v-model="form.is_active"
                    label="Aktif (sitede görünür)"
                    hide-details
                  />
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
              </VRow>
            </VForm>
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
      </VCol>
    </VRow>
  </div>
</template>
