<script setup lang="ts">
import type { ProductImage } from '~/types/api'

const props = defineProps<{
  images: ProductImage[]
  alt: string
}>()

const secili = ref(0)

// Ürün değişirse (aynı bileşen tekrar kullanılırsa) ilk görsele dön
watch(() => props.images, () => {
  secili.value = 0
})

const buyuk = computed(() => props.images?.[secili.value])
</script>

<template>
  <div>
    <!-- Büyük görsel. Sabit aspect-ratio: thumbnail değişince sayfa zıplamasın. -->
    <div
      class="w-full overflow-hidden rounded-lg bg-surface-container-low"
      style="aspect-ratio: 1 / 1"
    >
      <img
        v-if="buyuk"
        :src="buyuk.url_1200"
        :alt="alt"
        width="1200"
        height="1200"
        class="size-full object-cover"
      >
      <div v-else class="flex size-full items-center justify-center text-outline-variant">
        <Icon name="material-symbols:local-florist-outline" size="48" />
      </div>
    </div>

    <!-- Thumbnail'lar — tek görselli üründe gösterme, anlamsız -->
    <div v-if="images.length > 1" class="mt-4 flex gap-3">
      <button
        v-for="(img, i) in images"
        :key="i"
        type="button"
        class="size-20 overflow-hidden rounded-md border-2 transition-colors md:size-24"
        :class="i === secili ? 'border-secondary' : 'border-transparent hover:border-outline-variant'"
        :aria-label="`${i + 1}. görsel`"
        :aria-current="i === secili"
        @click="secili = i"
      >
        <img
          :src="img.url_400"
          :alt="`${alt} — ${i + 1}. görsel`"
          loading="lazy"
          width="400"
          height="400"
          class="size-full object-cover"
        >
      </button>
    </div>
  </div>
</template>
