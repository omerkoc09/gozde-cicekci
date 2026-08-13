<script setup lang="ts">
useSeoMeta({
  title: 'Ödeme Tamamlanamadı | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-20 text-center md:py-28">
    <span class="mx-auto flex size-16 items-center justify-center rounded-full bg-error/10">
      <Icon name="material-symbols:close" size="32" class="text-error" />
    </span>

    <h1 class="mt-8 font-serif text-3xl text-primary md:text-4xl">
      Ödeme Tamamlanamadı
    </h1>

    <p class="mx-auto mt-4 max-w-md text-body-lg text-on-surface-variant">
      Ödemeniz alınamadı. Sepetiniz korundu — dilerseniz tekrar deneyebilirsiniz.
    </p>

    <NuxtLink to="/siparis" class="btn-primary text-label-caps mt-10">
      Siparişe Dön
    </NuxtLink>
  </div>
</template>
