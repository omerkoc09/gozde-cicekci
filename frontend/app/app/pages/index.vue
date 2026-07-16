<script setup lang="ts">
const { data: featuredCategories } = await useFeaturedCategories()

// Ana sayfanın işi vitrin, katalog değil — 8 ürün yeter (spec §5.2)
const { data: products } = await useProductList({ limit: 8 })

useSeoMeta({
  title: 'Çiçekçi — Taze Çiçek ve Buket',
  description: 'Doğum günü, kutlama ve özel günler için taze çiçek ve buketler. Sipariş WhatsApp üzerinden.',
  ogTitle: 'Çiçekçi — Taze Çiçek ve Buket',
  ogDescription: 'Doğum günü, kutlama ve özel günler için taze çiçek ve buketler.',
  ogType: 'website',
})
</script>

<template>
  <div>
    <section class="hero">
      <div class="kapsayici">
        <h1>Taze çiçekler, aynı gün elinizde</h1>
        <p class="hero-metin">
          Doğum günü, kutlama ya da "sadece aklımdasın" demek için.
          Beğendiğiniz ürünü seçin, WhatsApp'tan tek mesajla sipariş verin.
        </p>
        <NuxtLink
          to="/urunler"
          class="buton"
        >
          Ürünleri Gör
        </NuxtLink>
      </div>
    </section>

    <!-- Esnaf hangi kategorilerin öne çıkacağını admin panelden seçiyor -->
    <section
      v-if="featuredCategories?.length"
      class="bolum"
    >
      <div class="kapsayici">
        <h2>Ne için arıyorsunuz?</h2>
        <div class="kategori-izgara">
          <NuxtLink
            v-for="kategori in featuredCategories"
            :key="kategori.id"
            :to="`/kategori/${kategori.slug}`"
            class="kategori-kart"
          >
            {{ kategori.name }}
          </NuxtLink>
        </div>
      </div>
    </section>

    <section class="bolum">
      <div class="kapsayici">
        <div class="baslik-satiri">
          <h2>Öne Çıkanlar</h2>
          <NuxtLink
            to="/urunler"
            class="tumu"
          >
            Tümünü gör →
          </NuxtLink>
        </div>

        <div
          v-if="products?.length"
          class="urun-izgara"
        >
          <ProductCard
            v-for="product in products"
            :key="product.id"
            :product="product"
          />
        </div>

        <p
          v-else
          class="soluk"
        >
          Ürünler yakında eklenecek.
        </p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.hero {
  padding-block: 2.5rem;
  background: var(--renk-zemin-alt);
  border-block-end: 1px solid var(--renk-cizgi);
}

.hero-metin {
  max-inline-size: 46ch;
  margin-block-end: 1.5rem;
  color: var(--renk-soluk);
}

.kategori-izgara {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--bosluk);
}

.kategori-kart {
  display: grid;
  place-items: center;
  min-block-size: 88px;
  padding: 1rem;
  border: 1px solid var(--renk-cizgi);
  border-radius: var(--yuvarlak);
  background: var(--renk-zemin-alt);
  font-weight: 600;
  text-align: center;
}

.kategori-kart:hover {
  border-color: var(--renk-vurgu);
  color: var(--renk-vurgu);
}

.baslik-satiri {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
}

.tumu {
  color: var(--renk-vurgu);
  font-weight: 500;
  white-space: nowrap;
}

@media (min-width: 640px) {
  .kategori-izgara { grid-template-columns: repeat(3, 1fr); }
}

@media (min-width: 768px) {
  .hero { padding-block: 4rem; }
}
</style>
