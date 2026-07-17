<script setup lang="ts">
const route = useRoute()
const orderNo = computed(() => String(route.query.no ?? ''))

// Sipariş no yoksa buraya doğrudan gelinmiş — ana sayfaya gönder
onMounted(() => {
  if (!orderNo.value)
    navigateTo('/')
})

useSeoMeta({
  title: 'Siparişiniz Alındı | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-20 text-center md:py-28">
    <span class="mx-auto flex size-16 items-center justify-center rounded-full bg-secondary/10">
      <Icon name="material-symbols:check" size="32" class="text-secondary" />
    </span>

    <h1 class="mt-8 font-serif text-3xl text-primary md:text-4xl">
      Siparişiniz Alındı
    </h1>

    <p v-if="orderNo" class="mt-4 text-body-lg text-on-surface-variant">
      Sipariş numaranız: <strong class="text-primary">{{ orderNo }}</strong>
    </p>

    <p class="mx-auto mt-4 max-w-md text-body-md text-on-surface-variant">
      En kısa sürede sizinle iletişime geçeceğiz. Sipariş numaranızı not
      almanızı öneririz.
    </p>

    <NuxtLink to="/urunler" class="btn-primary text-label-caps mt-10">
      Alışverişe Devam Et
    </NuxtLink>
  </div>
</template>
