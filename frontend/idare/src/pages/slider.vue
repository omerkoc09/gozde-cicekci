<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import { useSlides } from '@/composables/useSlides'
import type { Slide } from '@/model/slide'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'
import { requiredValidator } from '@validators'

const api = useSlides()

const loading = ref(false)
const slides = ref<Slide[]>([])

const dialog = ref(false)
const saving = ref(false)
const formRef = ref<VForm>()

// Düzenlenen slayt; null ise yeni kayıt.
const editing = ref<Slide | null>(null)

const form = ref({
  title: '',
  subtitle: '',
  is_active: true,
  sort_order: 0,
})

// Seçilen dosya. Yeni kayıtta zorunlu, düzenlemede boş bırakılırsa
// mevcut görsel korunur.
const imageFile = ref<File[]>([])

const secilenDosya = computed(() => imageFile.value[0])

// Yerel önizleme — dosya seçilince sunucuya gitmeden görünsün.
const preview = ref('')

watch(secilenDosya, file => {
  if (preview.value)
    URL.revokeObjectURL(preview.value)

  preview.value = file ? URL.createObjectURL(file) : ''
})

// Nesne URL'leri sızmasın.
onBeforeUnmount(() => {
  if (preview.value)
    URL.revokeObjectURL(preview.value)
})

const sirali = computed(() =>
  [...slides.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id))

const headers = [
  { title: 'Görsel', key: 'image', sortable: false, width: 120 },
  { title: 'Başlık', key: 'title' },
  { title: 'Alt Başlık', key: 'subtitle', sortable: false },
  { title: 'Aktif', key: 'is_active', sortable: false, width: 110 },
  { title: 'Sıra', key: 'sort_order', width: 90 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 110 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  slides.value = data ?? []
}

onMounted(load)

const openCreate = () => {
  editing.value = null
  form.value = { title: '', subtitle: '', is_active: true, sort_order: 0 }
  imageFile.value = []
  dialog.value = true
}

const openEdit = (s: Slide) => {
  editing.value = s
  form.value = {
    title: s.title,
    subtitle: s.subtitle,
    is_active: s.is_active,
    sort_order: s.sort_order,
  }
  imageFile.value = []
  dialog.value = true
}

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  // Yeni slaytta görsel zorunlu — backend de reddediyor ama kullanıcıyı
  // sunucuya kadar götürmeye gerek yok.
  if (!editing.value && !secilenDosya.value)
    return ErrorPopup('Slayt görseli seçilmeli')

  saving.value = true

  if (editing.value) {
    const [err] = await api.update(editing.value.id, { ...form.value })
    if (err) {
      saving.value = false

      return ErrorPopup(err.message)
    }

    // Görsel yalnızca yeni dosya seçildiyse değişir; metin güncellemesi
    // görsele dokunmaz.
    if (secilenDosya.value) {
      const [imgErr] = await api.replaceImage(editing.value.id, secilenDosya.value)
      if (imgErr) {
        saving.value = false

        return ErrorPopup(imgErr.message)
      }
    }
  }
  else {
    const [err] = await api.create({ ...form.value, image: secilenDosya.value })
    if (err) {
      saving.value = false

      return ErrorPopup(err.message)
    }
  }

  saving.value = false
  dialog.value = false
  SuccessToast(editing.value ? 'Slayt güncellendi' : 'Slayt eklendi')
  await load()
}

/** Switch anında kaydeder; yanıt gelene kadar satırdaki değer eski kalır. */
const toggle = async (s: Slide, value: boolean) => {
  const [err] = await api.update(s.id, { is_active: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

const remove = async (s: Slide) => {
  const ok = await ConfirmPopup(
    `"${s.title}" slaytı ve görseli silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [err] = await api.remove(s.id)
  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Slayt silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Slider Yönetimi
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="openCreate"
      >
        Yeni Slayt
      </VBtn>
    </div>

    <VAlert
      type="info"
      variant="tonal"
      class="mb-6"
    >
      Ana sayfadaki büyük görsel alanı. Slaytlar sıra numarasına göre gösterilir,
      otomatik olarak geçer; ziyaretçi oklarla da gezebilir. Pasif slayt
      görünmez. Hiç aktif slayt yoksa ana sayfa varsayılan görsele döner.
      <br>
      Önerilen görsel: en az 1920px genişlik, yatay (JPEG, PNG veya WebP; 25MB'a kadar).
      Küçük görsel yüklenirse büyütülmez — hero'da bulanık görünür.
    </VAlert>

    <VCard>
      <VDataTable
        :headers="headers"
        :items="sirali"
        :loading="loading"
        items-per-page="-1"
        no-data-text="Henüz slayt yok"
        loading-text="Yükleniyor..."
      >
        <template #item.image="{ item }">
          <VImg
            :src="item.url_400"
            :alt="item.title"
            width="96"
            height="54"
            cover
            class="rounded my-2"
          />
        </template>

        <template #item.title="{ item }">
          <span :class="{ 'text-disabled': !item.is_active }">{{ item.title }}</span>
        </template>

        <template #item.subtitle="{ item }">
          <span class="text-caption text-medium-emphasis">
            {{ item.subtitle || '—' }}
          </span>
        </template>

        <template #item.is_active="{ item }">
          <VSwitch
            :model-value="item.is_active"
            density="compact"
            hide-details
            @update:model-value="toggle(item, $event as boolean)"
          />
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
      max-width="560"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>{{ editing ? 'Slaytı Düzenle' : 'Yeni Slayt' }}</VCardTitle>
        </VCardItem>

        <VCardText>
          <VForm
            ref="formRef"
            @submit.prevent="save"
          >
            <VRow>
              <VCol cols="12">
                <VTextField
                  v-model="form.title"
                  label="Başlık"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <VCol cols="12">
                <VTextField
                  v-model="form.subtitle"
                  label="Alt Başlık"
                  hint="İsteğe bağlı"
                  persistent-hint
                />
              </VCol>

              <VCol cols="12">
                <VFileInput
                  v-model="imageFile"
                  label="Görsel"
                  accept="image/jpeg,image/png,image/webp"
                  prepend-icon=""
                  prepend-inner-icon="tabler-photo"
                  :hint="editing ? 'Boş bırakılırsa mevcut görsel korunur' : 'JPEG, PNG veya WebP — en az 1920px genişlik önerilir'"
                  persistent-hint
                />
              </VCol>

              <!-- Önizleme: yeni dosya seçildiyse onu, düzenlemede
                   seçilmediyse mevcut görseli göster. -->
              <VCol
                v-if="preview || editing"
                cols="12"
              >
                <VImg
                  :src="preview || editing?.url_1200"
                  :aspect-ratio="16 / 9"
                  cover
                  class="rounded border"
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
                  hint="Küçük olan önce gösterilir"
                  persistent-hint
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
