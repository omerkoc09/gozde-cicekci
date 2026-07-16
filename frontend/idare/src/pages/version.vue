<script setup lang="ts">
const version = import.meta.env.VITE_APP_VERSION;
</script>

<template>
  <div>
    {{ version}}
  </div>
</template>

<route lang="yaml">
meta:
  layout: blank
</route>
