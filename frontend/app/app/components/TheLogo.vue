<script setup lang="ts">
/**
 * Marka kilidi — logonun TEK kaynağı (spec §4.1).
 *
 * Oval "G" ikonu gerçek asset (referans_tasarım/logo/'dan alındı, viewBox
 * içeriğe oturtulup svgo ile optimize edildi). Marka adı ise canlı Libre Caslon
 * metni — asset olarak gömülmedi çünkü: her ekranda net, SEO'da okunur, ekran
 * okuyucu görür ve 0 KB.
 *
 * Not: Logo asset'i raster'dan trace edilmiş (bkz spec §4.1) — rengi
 * değiştirilemez, koyu zemine konamaz. Site baştan sona krem olduğu için sorun
 * değil. Orijinal vektör gelirse yalnızca bu dosya değişir.
 */
withDefaults(defineProps<{
  /** Tagline'ı gizle — dar alanlarda (mobil header, footer) */
  compact?: boolean
}>(), {
  compact: false,
})
</script>

<template>
  <NuxtLink
    to="/"
    class="group flex min-w-0 items-center gap-2.5 md:gap-5"
    aria-label="Gözde Tasarım Çiçekçilik — ana sayfa"
  >
    <img
      src="~/assets/img/gozde-icon.svg"
      alt=""
      aria-hidden="true"
      class="h-10 w-auto shrink-0 md:h-14"
      width="48"
      height="70"
    >

    <span class="flex min-w-0 flex-col border-l border-outline-variant/40 pl-2.5 md:pl-5">
      <span class="truncate font-serif text-sm leading-tight tracking-tight text-primary sm:text-base md:text-xl">
        Gözde Tasarım
      </span>
      <span class="-mt-0.5 truncate font-serif text-xs italic leading-none text-primary/80 sm:text-sm md:text-lg">
        Çiçekçilik
      </span>

      <span
        v-if="!compact"
        class="mt-1.5 hidden h-px w-10 bg-accent-gold md:block"
        aria-hidden="true"
      />
      <span
        v-if="!compact"
        class="mt-1.5 hidden text-[10px] font-semibold uppercase tracking-[0.28em] text-secondary md:block"
      >
        Çiçek • Çikolata • Hediyelik
      </span>
    </span>
  </NuxtLink>
</template>
