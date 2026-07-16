<script setup lang="ts">
/**
 * Boş durum — referans mockup'larda yok (hep dolu veri gösteriyorlar) ama
 * gerçek sitede kategori boş kalabilir, ürün eklenmemiş olabilir (spec §5.2).
 * Bunlar olmadan site premium değil yarım hisseder.
 */
withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
}>(), {
  icon: 'material-symbols:local-florist-outline',
  description: '',
})
</script>

<template>
  <div class="flex flex-col items-center justify-center px-6 py-20 text-center">
    <Icon :name="icon" size="40" class="text-outline-variant" />
    <p class="mt-5 font-serif text-xl text-primary">
      {{ title }}
    </p>
    <p v-if="description" class="mt-2 max-w-sm text-body-md text-on-surface-variant">
      {{ description }}
    </p>
    <slot />
  </div>
</template>
