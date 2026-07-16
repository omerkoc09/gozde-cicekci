<script setup lang="ts">
import banner from '~/assets/img/hesap-banner.webp'

/**
 * Hesap sayfası hero banner'ı — referansın "görsel arka planlı" varyantı
 * (spec §5.1). Düz başlıklı varyanttan belirgin şekilde daha premium.
 *
 * Not: Banner görseli referanstan alındı ama kırpıldı — orijinalinde başka bir
 * çiçekçinin site ekran görüntüsü vardı (markası ve menüsü görünüyordu).
 * Sadece çiçek bölgesi kullanıldı.
 */
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="relative isolate overflow-hidden rounded-lg">
    <img
      :src="banner"
      alt=""
      aria-hidden="true"
      class="absolute inset-0 -z-10 size-full object-cover object-right opacity-60"
      width="1600"
      height="891"
    >
    <div
      class="absolute inset-0 -z-10 bg-gradient-to-r from-surface via-surface/80 to-transparent"
      aria-hidden="true"
    />

    <div class="px-7 py-10 md:px-10 md:py-12">
      <h1 class="font-serif text-3xl text-primary md:text-4xl">
        {{ title }}
      </h1>
      <p v-if="description" class="mt-3 max-w-lg text-body-md text-on-surface-variant">
        {{ description }}
      </p>
    </div>
  </div>
</template>
