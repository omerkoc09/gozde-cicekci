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
const { data: products, status } = await useProductList({
  amac: category.value.axis === 'occasion' ? slug : undefined,
  tip: category.value.axis === 'type' ? slug : undefined,
  limit: 100,
})

// Bu sayfa indexlenir (filtre kombinasyonlarının aksine) — SEO'nun asıl hedefi.
// "geçmiş olsun çiçeği" arayan müşteri buraya düşer.
useSeoMeta({
  title: () => `${category.value?.name} | Gözde Tasarım Çiçekçilik`,
  description: () => `${category.value?.name} kategorisindeki özenle hazırlanmış taze çiçek tasarımları. Sipariş WhatsApp üzerinden.`,
  ogTitle: () => category.value?.name,
  ogType: 'website',
  robots: 'index, follow',
})
</script>

<template>
  <div v-if="category" class="site-container py-14 md:py-20">
    <BreadCrumb
      :items="[
        { label: 'Anasayfa', to: '/' },
        { label: 'Çiçekler', to: '/urunler' },
        { label: category.name },
      ]"
    />

    <h1 class="mt-6 font-serif text-4xl text-primary md:text-5xl">
      {{ category.name }}
    </h1>

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
      title="Bu kategoride şu an ürün yok"
      description="Diğer koleksiyonlarımıza göz atabilirsiniz."
    >
      <NuxtLink to="/urunler" class="btn-secondary text-label-caps mt-7">
        Tüm Koleksiyon
      </NuxtLink>
    </EmptyState>
  </div>
</template>
