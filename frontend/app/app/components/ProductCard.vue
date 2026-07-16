<script setup lang="ts">
import type { Product } from '~/types/api'
import { formatPrice } from '~/utils/price'

/**
 * Ürün kartı — DESIGN.md §Components: gölge YOK, hover'da ince border,
 * ürün adı serif, fiyat sans.
 */
const props = defineProps<{ product: Product }>()

/** Kapak = ilk görsel; backend sort_order'a göre sıralı döndürüyor (spec §4.4). */
const kapak = computed(() => props.product.images?.[0])
</script>

<template>
  <NuxtLink
    :to="`/urun/${product.slug}`"
    class="group block rounded-lg border border-transparent p-2 transition-colors hover:border-outline-variant/30"
  >
    <!-- Sabit aspect-ratio: fotoğraf inerken kart yüksekliği değişmesin,
         sayfa zıplamasın (spec §5.4). Referans 4:5 kullanıyor. -->
    <div class="relative mb-5 w-full overflow-hidden rounded-md bg-surface-container-low" style="aspect-ratio: 4 / 5">
      <!-- Backend hazır iki boyut veriyor (url_400 / url_1200) — IPX'ten
           tekrar geçirmeye gerek yok; kart için 400px kapak yeterli. -->
      <img
        v-if="kapak"
        :src="kapak.url_400"
        :alt="product.name"
        loading="lazy"
        width="400"
        height="500"
        class="size-full object-cover transition-transform duration-700 group-hover:scale-105"
      >
      <div
        v-else
        class="flex size-full items-center justify-center text-outline-variant"
      >
        <Icon name="material-symbols:local-florist-outline" size="40" />
      </div>
    </div>

    <div class="px-1 pb-2 text-center">
      <h3 class="line-clamp-2 font-serif text-base leading-snug text-primary">
        {{ product.name }}
      </h3>
      <p class="mt-2 text-body-md text-on-surface-variant">
        {{ formatPrice(product.price) }}
        <span class="text-xs text-on-surface-variant/70">(KDV dahil)</span>
      </p>
    </div>
  </NuxtLink>
</template>
