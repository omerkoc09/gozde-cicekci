<script setup lang="ts">
const route = useRoute()

// Filtre state'i URL'de (spec §5.6) — computed olarak okunuyor, useProductList
// reactive aldığı için query değişince liste kendiliğinden yenileniyor.
const amac = computed(() => route.query.amac as string | undefined)
const tip = computed(() => route.query.tip as string | undefined)

const { data: products } = await useProductList({
  amac,
  tip,
  limit: 100,
})

const filtreVar = computed(() => !!amac.value || !!tip.value)

// Filtre kombinasyonları noindex (spec §4.2): 10 occasion × 6 type = 60
// kombinasyon Google'da ince içerik üretir. Tekil kategoriler kendi
// path'lerinde indexleniyor (/kategori/[slug]).
useSeoMeta({
  title: 'Ürünler | Çiçekçi',
  description: 'Tüm çiçek ve buket çeşitleri. Gönderim amacına ve ürün tipine göre filtreleyin.',
  robots: () => (filtreVar.value ? 'noindex, follow' : 'index, follow'),
})
</script>

<template>
  <div class="kapsayici bolum">
    <h1>Ürünler</h1>

    <CategoryFilter />

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

    <div
      v-else
      class="bos"
    >
      <p>Bu filtreye uyan ürün bulunamadı.</p>
      <NuxtLink
        to="/urunler"
        class="buton"
      >
        Filtreyi temizle
      </NuxtLink>
    </div>
  </div>
</template>

<style scoped>
.liste {
  margin-block-start: 1.5rem;
}

.bos {
  margin-block-start: 2rem;
  padding: 2.5rem 1rem;
  text-align: center;
}

.bos p {
  margin-block-end: 1rem;
  color: var(--renk-soluk);
}
</style>
