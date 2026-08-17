<script setup lang="ts">
import { useProducts } from '@/composables/useProducts'
import type { Product, StokSebep } from '@/model/product'
import { ErrorPopup, SuccessToast } from '@/utils/Popup'

const props = defineProps<{
  modelValue: boolean
  product: Product | null

  /**
   * 'dusur' → stok azalt (WhatsApp satışı), 'artir' → stok ekle (yeni parti).
   * Yön dışarıdan geliyor: esnaf listede −/+ düğmesine basıyor, dialog
   * hangi işlemi yaptığını zaten biliyor.
   */
  yon: 'dusur' | 'artir'
}>()

const emit = defineEmits<{
  'update:modelValue': [boolean]
  'saved': []
}>()

const api = useProducts()

const adet = ref(1)
const sebep = ref<StokSebep>('whatsapp_satisi')
const indirimliSatis = ref(false)
const not = ref('')
const kaydediliyor = ref(false)

// Düşürmede WhatsApp satışı en sık kullanılan; artırmada yeni parti.
const dusurmeSebepleri = [
  { title: 'WhatsApp satışı', value: 'whatsapp_satisi' },
  { title: 'Sayım düzeltme', value: 'sayim_duzeltme' },
]

const artirmaSebepleri = [
  { title: 'Yeni parti', value: 'yeni_parti' },
  { title: 'Sayım düzeltme', value: 'sayim_duzeltme' },
]

const sebepler = computed(() =>
  props.yon === 'dusur' ? dusurmeSebepleri : artirmaSebepleri)

// İndirim aktifse (kota dolmamışsa) satışın kotayı tüketip tüketmediği
// sorulur — WhatsApp satışları da kotayı tüketiyor (spec §4.4).
const indirimAktif = computed(() => {
  const p = props.product
  if (!p || p.discount_price === null || p.discount_quota === null)
    return false

  return p.discount_sold < p.discount_quota
})

const mevcutStok = computed(() => props.product?.stock_quantity ?? 0)

// Düşürmede stoktan fazlası istenemez — sunucu da reddediyor ama
// kullanıcıya beklemeden söylemek daha iyi.
const adetGecerli = computed(() =>
  adet.value > 0 && (props.yon === 'artir' || adet.value <= mevcutStok.value))

watch(() => props.modelValue, acik => {
  if (!acik)
    return

  // Her açılışta sıfırla — önceki işlemin değerleri sızmasın.
  adet.value = 1
  not.value = ''
  sebep.value = props.yon === 'dusur' ? 'whatsapp_satisi' : 'yeni_parti'
  indirimliSatis.value = props.yon === 'dusur' && indirimAktif.value
})

const kapat = () => emit('update:modelValue', false)

const kaydet = async () => {
  if (!props.product || !adetGecerli.value)
    return

  kaydediliyor.value = true

  const [err] = await api.adjustStock(props.product.id, {
    delta: props.yon === 'dusur' ? -adet.value : adet.value,
    reason: sebep.value,
    was_discounted: indirimliSatis.value,
    note: not.value.trim(),
  })

  kaydediliyor.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Stok güncellendi')
  emit('saved')
  kapat()
}
</script>

<template>
  <VDialog
    :model-value="modelValue"
    max-width="480"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <VCard>
      <VCardTitle class="text-h6 pt-4">
        {{ yon === 'dusur' ? 'Stok Düşür' : 'Stok Ekle' }}
      </VCardTitle>

      <VCardText>
        <div class="text-body-2 text-medium-emphasis mb-4">
          {{ product?.name }} — mevcut stok: <strong>{{ mevcutStok }}</strong>
        </div>

        <VTextField
          v-model.number="adet"
          type="number"
          min="1"
          label="Adet"
          density="comfortable"
          :error-messages="adet > 0 && !adetGecerli ? `En fazla ${mevcutStok} adet düşülebilir` : ''"
        />

        <VSelect
          v-model="sebep"
          :items="sebepler"
          label="Sebep"
          density="comfortable"
          class="mt-3"
        />

        <VCheckbox
          v-if="yon === 'dusur' && indirimAktif"
          v-model="indirimliSatis"
          density="comfortable"
          hide-details
          class="mt-1"
        >
          <template #label>
            <span class="text-body-2">
              İndirimli satış
              <span class="text-medium-emphasis">
                (kalan indirimli adet: {{ (product?.discount_quota ?? 0) - (product?.discount_sold ?? 0) }})
              </span>
            </span>
          </template>
        </VCheckbox>

        <VTextField
          v-model="not"
          label="Not (isteğe bağlı)"
          density="comfortable"
          class="mt-3"
        />
      </VCardText>

      <VCardActions class="px-4 pb-4">
        <VSpacer />
        <VBtn
          variant="text"
          @click="kapat"
        >
          Vazgeç
        </VBtn>
        <VBtn
          color="primary"
          variant="flat"
          :loading="kaydediliyor"
          :disabled="!adetGecerli"
          @click="kaydet"
        >
          Kaydet
        </VBtn>
      </VCardActions>
    </VCard>
  </VDialog>
</template>
