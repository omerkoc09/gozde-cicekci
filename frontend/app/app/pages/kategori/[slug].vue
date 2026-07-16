<script setup lang="ts">
const route = useRoute()
const slug = String(route.params.slug)

const { data: category, error } = await useCategory(slug)

if (error.value || !category.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Kategori bulunamadı',
    fatal: true,
  })
}

// Kategori hangi eksende ise ürünler o parametreyle çekilir
const { data: products } = await useProductList({
  amac: category.value.axis === 'occasion' ? slug : undefined,
  tip: category.value.axis === 'type' ? slug : undefined,
  limit: 100,
})

// Bu sayfa indexlenir (filtre kombinasyonlarının aksine) — SEO'nun asıl hedefi.
// "geçmiş olsun çiçeği" arayan müşteri buraya düşer.
useSeoMeta({
  title: () => `${category.value?.name} | Çiçekçi`,
  description: () => `${category.value?.name} kategorisindeki taze çiçek ve buketler. WhatsApp'tan sipariş verin.`,
  ogTitle: () => category.value?.name,
  ogType: 'website',
  robots: 'index, follow',
})
</script>

<template>
  <div
    v-if="category"
    class="kapsayici bolum"
  >
    <nav class="izler soluk">
      <NuxtLink to="/">
        Ana Sayfa
      </NuxtLink>
      <span>/</span>
      <NuxtLink to="/urunler">
        Ürünler
      </NuxtLink>
    </nav>

    <h1>{{ category.name }}</h1>

    <div
      v-if="products?.length"
      class="urun-izgara liste"
    >
      <ProductCard
        v-for="product in products"
        :key="product.id"
        :product="product"
      />
    </div>

    <p
      v-else
      class="soluk bos"
    >
      Bu kategoride şu an ürün yok.
    </p>
  </div>
</template>

<style scoped>
.izler {
  display: flex;
  gap: 0.5rem;
  margin-block-end: 1rem;
  font-size: 0.9rem;
}

.izler a:hover {
  color: var(--renk-vurgu);
}

.liste {
  margin-block-start: 1.5rem;
}

.bos {
  margin-block-start: 2rem;
}
</style>
