<script setup lang="ts">
definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Favorilerim | Gözde Tasarım Çiçekçilik' })

/**
 * Favoriler INERT (spec §2.1) — backend'de favori yok.
 *
 * Referansta ürün fotoğrafı yerine yanlışlıkla *uygulama ekran görüntüleri*
 * konmuştu (spec §6.3); burada gerçek ürünler gösteriliyor. Kaydedilmiş
 * favori taklidi yapmak yerine katalogdan ilk birkaç ürünü "önizleme" olarak
 * listeliyoruz.
 */
const { data: products } = await useProductList({ limit: 4 })
</script>

<template>
  <div>
    <AccountHero
      title="Favoriler"
      description="Beğendiğiniz tasarımların özel koleksiyonu."
    />

    <div v-if="products?.length" class="mt-8 grid grid-cols-2 gap-3 md:gap-5 lg:grid-cols-3">
      <ProductCard
        v-for="product in products"
        :key="product.id"
        :product="product"
      />
    </div>

    <EmptyState
      v-else
      icon="material-symbols:favorite-outline"
      title="Henüz favoriniz yok"
      description="Beğendiğiniz tasarımları keşfedin."
    >
      <NuxtLink to="/urunler" class="btn-secondary text-label-caps mt-7">
        Koleksiyonu Keşfet
      </NuxtLink>
    </EmptyState>

    <AccountDemoNotice class="mt-10" />
  </div>
</template>
