<script setup lang="ts">
const sepetAcik = ref(false)
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-background">
    <TheHeader @open-cart="sepetAcik = true" />

    <main class="flex-1">
      <slot />
    </main>

    <TheFooter />

    <TheCartDrawer v-model="sepetAcik" />
    <WhatsAppFab />
  </div>
</template>
