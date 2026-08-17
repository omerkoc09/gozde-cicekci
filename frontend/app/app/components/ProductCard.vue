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

/**
 * İndirim yüzdesi eski/yeni fiyattan hesaplanır — ayrı alan tutulmuyor.
 * old_price yalnızca indirim aktifken doluyor (spec §3.3).
 */
const indirimYuzdesi = computed(() => {
  const eski = Number.parseFloat(props.product.old_price ?? '')
  const yeni = Number.parseFloat(props.product.price)

  if (Number.isNaN(eski) || Number.isNaN(yeni) || eski <= 0 || yeni >= eski)
    return 0

  return Math.round((1 - yeni / eski) * 100)
})

/** Tükendi rozeti — takipsiz ürün in_stock=true geldiği için hiç görünmez. */
const tukendi = computed(() => !props.product.in_stock)
</script>

<template>
  <NuxtLink
    :to="`/urun/${product.slug}`"
    class="group block rounded-lg border border-transparent p-2 transition-colors hover:border-outline-variant/30"
  >
    <!-- Sabit aspect-ratio: fotoğraf inerken kart yüksekliği değişmesin,
         sayfa zıplamasın (spec §5.4). Referans 4:5 kullanıyor. -->
    <div class="relative mb-5 w-full overflow-hidden rounded-md bg-surface-container-low" style="aspect-ratio: 4 / 5">
      <!-- Backend üç boyut veriyor (400/800/1200). Kart 4:5 dikey ve kaynak
           foto genelde yatay — object-cover kırpıp büyütünce 400px retina
           ekranda bulanık kalıyordu. srcset ile yüksek yoğunluklu ekran 800px
           yükler (kategori kartıyla aynı kalite). sizes: kart en fazla ~450px. -->
      <img
        v-if="kapak"
        :src="kapak.url_800"
        :srcset="`${kapak.url_400} 400w, ${kapak.url_800} 800w`"
        sizes="(max-width: 640px) 50vw, 320px"
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

      <!-- İki rozet bağımsız, aynı anda görünebilir: karşı köşelere
           konumlandırıldılar ki üst üste binmesinler (spec §6.1). -->
      <span
        v-if="indirimYuzdesi > 0"
        class="absolute left-2 top-2 rounded bg-primary px-2 py-1 text-xs font-medium tracking-wide text-on-primary"
      >
        %{{ indirimYuzdesi }} İNDİRİM
      </span>

      <span
        v-if="tukendi"
        class="absolute right-2 top-2 rounded bg-surface/95 px-2 py-1 text-xs font-medium tracking-wide text-on-surface-variant"
      >
        TÜKENDİ
      </span>

      <!-- Tükenen ürün soluk gösterilir ama gizlenmez — müşteri görüp
           WhatsApp'tan sorabilsin (spec §6.1). -->
      <div
        v-if="tukendi"
        class="pointer-events-none absolute inset-0 bg-surface/45"
      />
    </div>

    <div class="px-1 pb-2 text-center">
      <h3 class="line-clamp-2 font-serif text-base leading-snug text-primary">
        {{ product.name }}
      </h3>
      <p class="mt-2 text-body-md text-on-surface-variant">
        <!-- Eski fiyat üstü çizili, indirimli fiyat vurgulu — ikisi de
             görünür olmalı (spec §6.1). -->
        <span
          v-if="product.old_price"
          class="me-1.5 text-sm text-on-surface-variant/60 line-through"
        >
          {{ formatPrice(product.old_price) }}
        </span>
        <span :class="product.old_price ? 'font-medium text-primary' : ''">
          {{ formatPrice(product.price) }}
        </span>
        <span class="text-xs text-on-surface-variant/70">(KDV dahil)</span>
      </p>
    </div>
  </NuxtLink>
</template>
