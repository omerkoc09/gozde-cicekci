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
    <div class="buyuk">
      <NuxtImg
        v-if="buyuk"
        :src="buyuk.url_1200"
        :alt="alt"
        loading="eager"
        width="1200"
        height="1200"
        sizes="100vw md:600px"
      />
      <!-- Çiçekçi henüz fotoğraf yüklememiş olabilir — sayfa patlamamalı -->
      <div
        v-else
        class="gorsel-yok"
      >
        🌸
      </div>
    </div>

    <!-- Tek görselde şerit gösterme -->
    <div
      v-if="images.length > 1"
      class="serit"
    >
      <button
        v-for="(img, i) in images"
        :key="img.url_400"
        class="serit-oge"
        :class="{ 'serit-secili': i === secili }"
        :aria-label="`${i + 1}. fotoğraf`"
        @click="secili = i"
      >
        <NuxtImg
          :src="img.url_400"
          :alt="`${alt} — ${i + 1}. fotoğraf`"
          loading="lazy"
          width="400"
          height="400"
        />
      </button>
    </div>
  </div>
</template>

<style scoped>
.buyuk {
  aspect-ratio: 1;
  inline-size: 100%;
  border-radius: var(--yuvarlak);
  overflow: hidden;
  background: var(--renk-zemin-alt);
}

.buyuk :deep(img) {
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.serit {
  display: flex;
  gap: 0.5rem;
  margin-block-start: 0.6rem;
  overflow-x: auto;
}

.serit-oge {
  flex: 0 0 68px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 8px;
  overflow: hidden;
  background: none;
  cursor: pointer;
  line-height: 0;
}

.serit-secili {
  border-color: var(--renk-vurgu);
}

.serit-oge :deep(img) {
  inline-size: 100%;
  aspect-ratio: 1;
  object-fit: cover;
}
</style>
