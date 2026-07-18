<script setup lang="ts">
const route = useRoute()
const { clear } = useCart()
const orderNo = computed(() =>
  String(route.query.no ?? (import.meta.client ? sessionStorage.getItem('bekleyen_siparis_no') : '') ?? ''))

// Sipariş no yoksa buraya doğrudan gelinmiş — ana sayfaya gönder
onMounted(() => {
  if (!orderNo.value) {
    navigateTo('/')
    return
  }
  clear() // ödeme başarılı döndü, sepeti temizle
  sessionStorage.removeItem('bekleyen_siparis_no')
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

    <p class="mt-4 text-body-lg text-on-surface-variant">
      Ödemeniz alındı, siparişiniz hazırlanıyor.
      <span v-if="orderNo">Sipariş numaranız: <strong class="text-primary">{{ orderNo }}</strong></span>
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
