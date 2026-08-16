<script setup lang="ts">
import type { CartItemOption, ProductOptionGroup } from '~/types/api'

const props = defineProps<{
  groups: ProductOptionGroup[]
  modelValue: CartItemOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [CartItemOption[]]
}>()

const secilenId = (groupId: number) => {
  const g = props.groups.find(x => x.id === groupId)

  return props.modelValue.find(o => g?.values.some(v => v.id === o.value_id))?.value_id
}

/** Seçilen değerin adı — başlığın yanında gösterilir (renk noktası küçük). */
const secilenAd = (groupId: number) => {
  const g = props.groups.find(x => x.id === groupId)

  return props.modelValue.find(o => g?.values.some(v => v.id === o.value_id))?.value_name
}

const sec = (group: ProductOptionGroup, valueId: number) => {
  const deger = group.values.find(v => v.id === valueId)
  if (!deger)
    return

  // Aynı gruptan önceki seçim düşer — grup başına tek değer.
  const digerGruplar = props.modelValue.filter(
    o => !group.values.some(v => v.id === o.value_id))

  emit('update:modelValue', [...digerGruplar, {
    value_id: deger.id,
    group_name: group.name,
    value_name: deger.name,
    swatch_hex: deger.swatch_hex,
  }])
}

// Eksik zorunlu grupların hesabı BİLEREK burada değil: "Sepete Ekle"
// butonu ve uyarı metni ürün sayfasında, hesap da orada yapılıyor
// (urun/[slug].vue). Bileşene ref'le erişip expose edilen bir computed
// okumak gereksiz dolaylılık olurdu.
</script>

<template>
  <div class="space-y-6">
    <div
      v-for="g in groups"
      :key="g.id"
    >
      <!--
        Başlık sitenin küçük etiket tipografisiyle (uppercase + geniş
        letter-spacing) — kategori/bölüm başlıklarıyla aynı dil.
        Seçilen değerin adı başlığın yanında görünür: renk noktası
        küçüldüğü için hangi rengin seçildiği yazıyla da okunmalı.
      -->
      <p class="mb-3 flex items-baseline gap-2">
        <span class="text-label-caps text-secondary">{{ g.name }}</span>
        <span
          v-if="secilenAd(g.id)"
          class="text-body-sm text-on-surface-variant"
        >{{ secilenAd(g.id) }}</span>
      </p>

      <!--
        Renk tek başına bilgi taşımamalı: her seçenek aria-label ile
        rengin ADINI da taşıyor, seçili olan aria-checked ile bildiriliyor.
      -->
      <div
        role="radiogroup"
        :aria-label="g.name"
        class="flex flex-wrap"
        :class="g.kind === 'color' ? 'gap-2.5' : 'gap-2'"
      >
        <!--
          Renk seçimi: nokta 22px, çevresinde ince ayrım halkası (açık
          renkler beyaz zeminde kaybolmasın). Seçim, dışarıdan geçen ikinci
          bir halkayla belirtiliyor — noktanın kendi boyutu değişmediği için
          satır zıplamıyor.
        -->
        <button
          v-for="v in g.values"
          :key="v.id"
          type="button"
          role="radio"
          :aria-checked="secilenId(g.id) === v.id"
          :aria-label="v.name"
          :title="v.name"
          class="cursor-pointer rounded-full transition-all duration-200"
          :class="g.kind === 'color'
            ? 'size-5.5 ring-1 ring-black/10 hover:scale-110'
            : [
              'text-body-sm rounded-full border px-3.5 py-1 transition-colors',
              secilenId(g.id) === v.id
                ? 'border-secondary bg-secondary/10 text-secondary'
                : 'border-outline-variant text-on-surface-variant hover:border-secondary/60',
            ]"
          :style="g.kind === 'color'
            ? {
              background: v.swatch_hex,
              boxShadow: secilenId(g.id) === v.id
                ? '0 0 0 2px var(--color-surface), 0 0 0 3.5px var(--color-secondary)'
                : undefined,
            }
            : undefined"
          @click="sec(g, v.id)"
        >
          <span v-if="g.kind !== 'color'">{{ v.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
