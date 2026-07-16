<script setup lang="ts">
import { useImages } from '@/composables/useImages'
import type { ProductImage } from '@/model/product'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'

const props = defineProps<{
  productId: number
  images: ProductImage[]
}>()

const emit = defineEmits<{
  update: []
}>()

const api = useImages()

const uploading = ref(false)
const busy = ref(false)
const files = ref<File[]>([])

// sort_order'a göre sıralı — ilki kapak (spec §4.4).
const ordered = computed(() =>
  [...props.images].sort((a, b) => a.sort_order - b.sort_order))

const upload = async () => {
  const list = files.value
  if (!list.length)
    return

  uploading.value = true

  for (const file of list) {
    const [err] = await api.upload(props.productId, file)
    if (err) {
      uploading.value = false
      files.value = []
      ErrorPopup(err.message)
      emit('update')

      return
    }
  }

  uploading.value = false
  files.value = []
  SuccessToast(list.length > 1 ? 'Görseller yüklendi' : 'Görsel yüklendi')
  emit('update')
}

/**
 * Görseli bir sıra kaydırır. Backend TÜM görsellerin id'sini istiyor —
 * yeni sıralamanın tamamı gönderiliyor, ilk id kapak oluyor.
 */
const move = async (index: number, direction: -1 | 1) => {
  const next = index + direction
  if (next < 0 || next >= ordered.value.length)
    return

  const ids = ordered.value.map(i => i.id)

  ;[ids[index], ids[next]] = [ids[next], ids[index]]

  busy.value = true

  const [err] = await api.reorder(props.productId, ids)

  busy.value = false

  if (err)
    return ErrorPopup(err.message)

  emit('update')
}

const remove = async (img: ProductImage) => {
  const ok = await ConfirmPopup('Bu görsel silinecek. Devam edilsin mi?', 'Sil', 'Vazgeç')
  if (!ok)
    return

  busy.value = true

  const [err] = await api.remove(img.id)

  busy.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Görsel silindi')
  emit('update')
}
</script>

<template>
  <div>
    <VFileInput
      v-model="files"
      label="Fotoğraf seç"
      accept="image/jpeg,image/png"
      prepend-icon="tabler-camera"
      multiple
      chips
      :disabled="uploading || busy"
      :loading="uploading"
      hint="JPEG veya PNG. İlk görsel kapak olarak kullanılır."
      persistent-hint
      class="mb-4"
      @update:model-value="upload"
    />

    <VProgressLinear
      v-if="uploading"
      indeterminate
      class="mb-4"
    />

    <VAlert
      v-if="!ordered.length && !uploading"
      type="info"
      variant="tonal"
      density="compact"
    >
      Henüz görsel yok. Eklediğiniz ilk görsel kapak olur.
    </VAlert>

    <VRow v-else>
      <VCol
        v-for="(img, i) in ordered"
        :key="img.id"
        cols="12"
        sm="6"
      >
        <VCard variant="outlined">
          <VImg
            :src="img.url_400"
            aspect-ratio="1"
            cover
          >
            <!-- Kapak = ilk görsel (spec §4.4) -->
            <VChip
              v-if="i === 0"
              size="x-small"
              color="primary"
              variant="elevated"
              class="ma-2"
            >
              Kapak
            </VChip>
          </VImg>

          <div class="d-flex align-center justify-space-between pa-1">
            <div class="d-flex">
              <VBtn
                icon="tabler-arrow-left"
                variant="text"
                size="x-small"
                :disabled="i === 0 || busy"
                @click="move(i, -1)"
              />
              <VBtn
                icon="tabler-arrow-right"
                variant="text"
                size="x-small"
                :disabled="i === ordered.length - 1 || busy"
                @click="move(i, 1)"
              />
            </div>

            <VBtn
              icon="tabler-trash"
              variant="text"
              size="x-small"
              color="error"
              :disabled="busy"
              @click="remove(img)"
            />
          </div>
        </VCard>
      </VCol>
    </VRow>
  </div>
</template>
