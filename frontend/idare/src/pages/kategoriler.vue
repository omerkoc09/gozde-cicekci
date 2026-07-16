<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import { useCategories } from '@/composables/useCategories'
import type { Axis, Category } from '@/model/category'
import { AXIS_LABELS } from '@/model/category'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'
import { requiredValidator } from '@validators'

const api = useCategories()

const loading = ref(false)
const categories = ref<Category[]>([])

const dialog = ref(false)
const saving = ref(false)
const formRef = ref<VForm>()

// Düzenlenen kategori; null ise yeni kayıt.
const editing = ref<Category | null>(null)

const form = ref({
  name: '',
  axis: 'occasion' as Axis,
  is_active: true,
  is_featured: false,
  sort_order: 0,
})

const byAxis = (axis: Axis) =>
  categories.value
    .filter(c => c.axis === axis)
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'tr'))

const occasionCategories = computed(() => byAxis('occasion'))
const typeCategories = computed(() => byAxis('type'))

const headers = [
  { title: 'Ad', key: 'name' },
  { title: 'Link (slug)', key: 'slug', sortable: false },
  { title: 'Aktif', key: 'is_active', sortable: false, width: 110 },
  { title: 'Öne Çıkan', key: 'is_featured', sortable: false, width: 130 },
  { title: 'Sıra', key: 'sort_order', width: 90 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 110 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  categories.value = data ?? []
}

onMounted(load)

const openCreate = () => {
  editing.value = null
  form.value = { name: '', axis: 'occasion', is_active: true, is_featured: false, sort_order: 0 }
  dialog.value = true
}

const openEdit = (c: Category) => {
  editing.value = c
  form.value = {
    name: c.name,
    axis: c.axis,
    is_active: c.is_active,
    is_featured: c.is_featured,
    sort_order: c.sort_order,
  }
  dialog.value = true
}

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  saving.value = true

  // axis yalnızca oluştururken gönderilir — değiştirilemez (spec §4.1).
  const [err] = editing.value
    ? await api.update(editing.value.id, {
      name: form.value.name,
      is_active: form.value.is_active,
      is_featured: form.value.is_featured,
      sort_order: form.value.sort_order,
    })
    : await api.create({ ...form.value })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  dialog.value = false
  SuccessToast(editing.value ? 'Kategori güncellendi' : 'Kategori oluşturuldu')
  await load()
}

/**
 * Switch'ler anında kaydeder. İyimser değil: yanıt gelene kadar satırdaki
 * değer eski kalır, hata olursa hiç değişmez.
 */
const toggle = async (c: Category, field: 'is_active' | 'is_featured', value: boolean) => {
  const [err] = await api.update(c.id, { [field]: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

const remove = async (c: Category) => {
  const [err, data] = await api.productCount(c.id)
  if (err)
    return ErrorPopup(err.message)

  const n = data.product_count

  const ok = await ConfirmPopup(
    n > 0
      ? `"${c.name}" kategorisinde ${n} ürün var. Silerseniz bu ürünler kategoriden çıkacak (ürünler silinmez). Devam edilsin mi?`
      : `"${c.name}" kategorisi silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [delErr] = await api.remove(c.id)
  if (delErr)
    return ErrorPopup(delErr.message)

  SuccessToast('Kategori silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Kategoriler
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="openCreate"
      >
        Yeni Kategori
      </VBtn>
    </div>

    <VAlert
      type="info"
      variant="tonal"
      class="mb-6"
    >
      Pasif kategori ana sayfada görünmez — öne çıkarılmış olsa bile.
      Kategori linki (slug) oluşturulduktan sonra değişmez.
    </VAlert>

    <VCard
      v-for="axis in (['occasion', 'type'] as Axis[])"
      :key="axis"
      class="mb-6"
    >
      <VCardItem>
        <VCardTitle>{{ AXIS_LABELS[axis] }}</VCardTitle>
      </VCardItem>

      <VDataTable
        :headers="headers"
        :items="axis === 'occasion' ? occasionCategories : typeCategories"
        :loading="loading"
        items-per-page="-1"
        no-data-text="Bu eksende henüz kategori yok"
        loading-text="Yükleniyor..."
      >
        <template #item.name="{ item }">
          <span :class="{ 'text-disabled': !item.is_active }">{{ item.name }}</span>
        </template>

        <template #item.slug="{ item }">
          <code class="text-caption text-medium-emphasis">{{ item.slug }}</code>
        </template>

        <template #item.is_active="{ item }">
          <VSwitch
            :model-value="item.is_active"
            density="compact"
            hide-details
            @update:model-value="toggle(item, 'is_active', $event as boolean)"
          />
        </template>

        <template #item.is_featured="{ item }">
          <VTooltip
            :disabled="item.is_active"
            text="Pasif kategori öne çıkarılamaz"
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
                  @update:model-value="toggle(item, 'is_featured', $event as boolean)"
                />
              </div>
            </template>
          </VTooltip>
        </template>

        <template #item.sort_order="{ item }">
          <span :class="{ 'text-disabled': !item.is_active }">{{ item.sort_order }}</span>
        </template>

        <template #item.actions="{ item }">
          <VBtn
            icon="tabler-pencil"
            variant="text"
            size="small"
            @click="openEdit(item)"
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

    <!-- 👉 Oluştur / Düzenle -->
    <VDialog
      v-model="dialog"
      max-width="520"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>{{ editing ? 'Kategoriyi Düzenle' : 'Yeni Kategori' }}</VCardTitle>
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
                  label="Kategori Adı"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <VCol cols="12">
                <!--
                  Eksen yalnızca oluştururken seçilir; sonradan
                  değiştirilemez çünkü ürün ilişkileri anlamsızlaşır.
                -->
                <VRadioGroup
                  v-if="!editing"
                  v-model="form.axis"
                  label="Eksen"
                  inline
                >
                  <VRadio
                    label="Gönderim Amacı"
                    value="occasion"
                  />
                  <VRadio
                    label="Ürün Tipi"
                    value="type"
                  />
                </VRadioGroup>

                <VTextField
                  v-else
                  :model-value="AXIS_LABELS[form.axis]"
                  label="Eksen"
                  readonly
                  disabled
                  hint="Eksen değiştirilemez"
                  persistent-hint
                />
              </VCol>

              <VCol
                cols="12"
                sm="6"
              >
                <VTextField
                  v-model.number="form.sort_order"
                  label="Sıra"
                  type="number"
                />
              </VCol>

              <VCol
                cols="12"
                sm="6"
              >
                <VSwitch
                  v-model="form.is_active"
                  label="Aktif"
                  hide-details
                />
                <VSwitch
                  v-model="form.is_featured"
                  label="Öne Çıkan"
                  :disabled="!form.is_active"
                  hide-details
                />
              </VCol>
            </VRow>
          </VForm>
        </VCardText>

        <VCardActions class="px-6 pb-4">
          <VSpacer />
          <VBtn
            variant="text"
            :disabled="saving"
            @click="dialog = false"
          >
            Vazgeç
          </VBtn>
          <VBtn
            :loading="saving"
            @click="save"
          >
            Kaydet
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </div>
</template>
