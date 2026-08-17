<script setup lang="ts">
/**
 * İndirimli Ürünler — TÜRETİLMİŞ liste (spec §6.2).
 *
 * Gerçek bir kategori değil: categories tablosuna kayıt eklenmiyor.
 * Esnaf ürüne indirim girdiği an burada belirir, kota dolunca kendiliğinden
 * çıkar. Senkron tutulacak ikinci bir yer yok.
 */
const { data: products, status } = await useProductList({
  indirimli: true,
  limit: 100,
})

useSeoMeta({
  title: 'İndirimli Ürünler | Gözde Tasarım Çiçekçilik',
  description: 'Sınırlı sayıda indirimli çiçek ve buket tasarımları. Kampanya adedi dolduğunda indirim sona erer.',
  ogTitle: 'İndirimli Ürünler',
  ogType: 'website',
  robots: 'index, follow',
})
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <BreadCrumb
      :items="[
        { label: 'Anasayfa', to: '/' },
        { label: 'İndirimli Ürünler' },
      ]"
    />

    <h1 class="mt-6 font-serif text-4xl text-primary md:text-5xl">
      İndirimli Ürünler
    </h1>

    <p class="mt-3 text-body-md text-on-surface-variant">
      Sınırlı sayıda ürün indirimli fiyattan satılıyor. Belirtilen adet
      tükendiğinde indirim sona erer.
    </p>

    <div v-if="status === 'pending'" class="mt-12 grid grid-cols-2 gap-3 md:gap-6 lg:grid-cols-4">
      <ProductCardSkeleton v-for="i in 8" :key="i" />
    </div>

    <div v-else-if="products?.length" class="mt-12 grid grid-cols-2 gap-3 md:gap-6 lg:grid-cols-4">
      <ProductCard
        v-for="product in products"
        :key="product.id"
        :product="product"
      />
    </div>

    <EmptyState
      v-else
      title="Şu anda indirimli ürün bulunmuyor"
      description="Yeni kampanyalar için takipte kalın."
    >
      <NuxtLink to="/urunler" class="btn-secondary text-label-caps mt-7">
        Tüm Koleksiyon
      </NuxtLink>
    </EmptyState>
  </div>
</template>
