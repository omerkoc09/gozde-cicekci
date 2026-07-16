<script setup lang="ts">
/**
 * Ürün detaydaki açılır bölümler (Teslimat Bilgileri, İade ve Değişim).
 *
 * <details> üzerine kurulu: JS olmadan da açılır, klavye ve ekran okuyucu
 * desteği tarayıcıdan bedava gelir.
 */
defineProps<{ title: string }>()
</script>

<template>
  <details class="group border-b border-outline-variant/40">
    <summary
      class="flex cursor-pointer list-none items-center justify-between py-4 text-body-md text-on-surface marker:hidden"
    >
      {{ title }}
      <Icon
        name="material-symbols:keyboard-arrow-down"
        size="20"
        class="shrink-0 text-on-surface-variant transition-transform duration-200 group-open:rotate-180"
      />
    </summary>
    <div class="pb-5 pr-6 text-body-md text-on-surface-variant">
      <slot />
    </div>
  </details>
</template>

<style scoped>
/* Safari <summary> varsayılan üçgenini gizler */
summary::-webkit-details-marker {
  display: none;
}
</style>
