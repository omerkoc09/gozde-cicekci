<script setup lang="ts">
const sepetAcik = ref(false)

// Sayfalar da sepeti açabilsin (ürün detaydaki inert "Sepete Ekle" butonu).
provide('sepetAcik', sepetAcik)
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-background">
    <TheHeader @open-cart="sepetAcik = true" />

    <main class="flex-1 pb-16 lg:pb-0">
      <slot />
    </main>

    <TheFooter />

    <TheCartDrawer v-model="sepetAcik" />
    <WhatsAppFab />
    <TheBottomNav @open-cart="sepetAcik = true" />
  </div>
</template>
