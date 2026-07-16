<script setup lang="ts">
/**
 * İz yolu — referansta ürün detayda var (ANASAYFA › GÜLLER › 15'Lİ...).
 * label-caps tipografisi, ayraç olarak "›".
 */
defineProps<{
  items: Array<{ label: string, to?: string }>
}>()
</script>

<template>
  <nav aria-label="İz yolu">
    <ol class="text-label-caps flex flex-wrap items-center gap-2 text-on-surface-variant/70">
      <li v-for="(item, i) in items" :key="i" class="flex items-center gap-2">
        <NuxtLink
          v-if="item.to"
          :to="item.to"
          class="transition-colors hover:text-secondary"
        >
          {{ item.label }}
        </NuxtLink>
        <span v-else class="text-on-surface-variant" aria-current="page">
          {{ item.label }}
        </span>

        <span v-if="i < items.length - 1" aria-hidden="true">›</span>
      </li>
    </ol>
  </nav>
</template>
