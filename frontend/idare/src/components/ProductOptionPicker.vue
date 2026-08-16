<script setup lang="ts">
import { useOptions } from '@/composables/useOptions'
import type { OptionGroup } from '@/model/option'
import type { ProductOptionGroupLink } from '@/model/product'
import { ErrorPopup } from '@/utils/Popup'

const props = defineProps<{

  /** Ürünün mevcut bağları; yeni üründe boş dizi. */
  modelValue: ProductOptionGroupLink[]
}>()

const emit = defineEmits<{
  'update:modelValue': [ProductOptionGroupLink[]]
}>()

const api = useOptions()

const gruplar = ref<OptionGroup[]>([])
const loading = ref(false)

// Yalnızca aktif ve en az bir değeri olan gruplar seçilebilir — değeri
// olmayan grubu ürüne açmak müşteriye boş bir başlık gösterirdi.
const secilebilir = computed(() =>
  gruplar.value.filter(g => g.is_active && g.values.some(v => v.is_active)))

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  gruplar.value = data ?? []
}

onMounted(load)

const bagliMi = (groupId: number) =>
  props.modelValue.some(l => l.group_id === groupId)

const toggleGrup = (groupId: number, acik: boolean) => {
  emit('update:modelValue', acik
    ? [...props.modelValue, { group_id: groupId }]
    : props.modelValue.filter(l => l.group_id !== groupId))
}
</script>

<template>
  <div>
    <p class="text-subtitle-1 mb-1">
      Özelleştirme
    </p>
    <p class="text-caption text-medium-emphasis mb-4">
      İşaretlenen gruplar müşteriye ürün sayfasında sorulur. Müşteri hiçbir
      şey seçmezse listedeki ilk renk geçerli olur — sipariş her zaman bir
      seçimle gelir.
    </p>

    <VProgressLinear
      v-if="loading"
      indeterminate
      class="mb-4"
    />

    <VAlert
      v-else-if="!secilebilir.length"
      type="info"
      variant="tonal"
      density="compact"
    >
      Henüz kullanılabilir seçenek grubu yok. Seçenek Yönetimi sayfasından
      grup ve renk ekleyebilirsiniz.
    </VAlert>

    <div
      v-for="g in secilebilir"
      :key="g.id"
      class="d-flex align-center flex-wrap ga-4 py-2 border-b"
    >
      <VCheckbox
        :model-value="bagliMi(g.id)"
        :label="g.name"
        density="compact"
        hide-details
        style="min-inline-size: 200px;"
        @update:model-value="toggleGrup(g.id, $event as boolean)"
      />

      <!--
        Grubun renkleri — salt okunur önizleme, esnaf ne sunulacağını görsün.
        İlki müşteriye varsayılan olarak seçili gelir.
      -->
      <div class="d-flex ga-1 flex-wrap">
        <VTooltip
          v-for="v in g.values.filter(x => x.is_active)"
          :key="v.id"
          :text="v.name"
          location="top"
        >
          <template #activator="{ props: tip }">
            <span
              v-if="g.kind === 'color'"
              v-bind="tip"
              class="d-inline-block rounded-circle border"
              :style="{ background: v.swatch_hex, inlineSize: '18px', blockSize: '18px' }"
            />
            <VChip
              v-else
              v-bind="tip"
              size="x-small"
              variant="tonal"
            >
              {{ v.name }}
            </VChip>
          </template>
        </VTooltip>
      </div>
    </div>
  </div>
</template>
