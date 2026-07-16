<script setup lang="ts">
import type { Product } from '~/types/api'
import { formatPrice } from '~/utils/price'

const props = defineProps<{ product: Product }>()

/** Kapak = ilk görsel; backend sort_order'a göre sıralı döndürüyor (spec §4.4). */
const kapak = computed(() => props.product.images?.[0])
</script>

<template>
  <NuxtLink
    :to="`/urun/${product.slug}`"
    class="kart"
  >
    <!-- Sabit aspect-ratio: fotoğraf inerken kart yüksekliği değişmesin,
         sayfa zıplamasın (spec §5.4). -->
    <div class="kart-gorsel">
      <NuxtImg
        v-if="kapak"
        :src="kapak.url_400"
        :alt="product.name"
        loading="lazy"
        width="400"
        height="400"
        sizes="50vw sm:33vw md:25vw"
      />
      <div
        v-else
        class="gorsel-yok"
      >
        🌸
      </div>
    </div>

    <div class="kart-alt">
      <h3 class="kart-ad">
        {{ product.name }}
      </h3>
      <p class="kart-fiyat">
        {{ formatPrice(product.price) }}
      </p>
    </div>
  </NuxtLink>
</template>

<style scoped>
.kart {
  display: block;
  border: 1px solid var(--renk-cizgi);
  border-radius: var(--yuvarlak);
  overflow: hidden;
  background: var(--renk-zemin);
  transition: box-shadow 0.15s;
}

.kart:hover {
  box-shadow: 0 4px 16px rgb(0 0 0 / 8%);
}

.kart-gorsel {
  aspect-ratio: 1;
  inline-size: 100%;
  background: var(--renk-zemin-alt);
}

.kart-gorsel :deep(img) {
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.kart-alt {
  padding: 0.7rem;
}

.kart-ad {
  margin: 0 0 0.25rem;
  font-size: 0.95rem;
  font-weight: 500;

  /* Uzun ürün adı kartı bozmasın */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.kart-fiyat {
  margin: 0;
  font-weight: 700;
  color: var(--renk-vurgu);
}
</style>
