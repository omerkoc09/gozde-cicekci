<template>
  <div class="duzen">
    <TheHeader />

    <main>
      <slot />
    </main>

    <TheFooter />
  </div>
</template>

<style scoped>
/* Footer içerik kısa olsa da altta kalsın */
.duzen {
  display: flex;
  flex-direction: column;
  min-block-size: 100vh;
}

main {
  flex: 1;
}
</style>
