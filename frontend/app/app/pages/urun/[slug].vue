<script setup lang="ts">
import { formatPrice } from '~/utils/price'

const route = useRoute()
const slug = String(route.params.slug)

const { data: product, error } = await useProduct(slug)

if (error.value || !product.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Ürün bulunamadı',
    fatal: true,
  })
}

// ⚠️ Slug geçmişi 301'i (spec §4.2)
// Backend eski slug'a 301 döner ama Location "/api/products/..." — yani API
// yolu, sayfa yolu değil. useFetch bu 301'i şeffafça takip edip ürünü getirir,
// ama tarayıcının adresi eski slug'da kalır. Sonuç: Google iki URL'de aynı
// içeriği görür ve slug geçmişinin amacı boşa gider.
// Çözüm: yanıttaki slug istenen slug'dan farklıysa SAYFA yolunda 301 yap.
if (product.value.slug !== slug) {
  await navigateTo(`/urun/${product.value.slug}`, {
    redirectCode: 301,
    replace: true,
  })
}

// Ürün yanıtında kategori İSİMLERİ yok, sadece category_ids — isimler için
// tüm kategori listesi çekiliyor (~16 kayıt, tek çağrı).
const { data: categories } = await useCategoryList()

const productCategories = computed(() =>
  (categories.value ?? []).filter(c => product.value?.category_ids?.includes(c.id)))

const { public: cfg } = useRuntimeConfig()

// WhatsApp önizlemesinin çalıştığı yer — og:image SSR'da gelmek zorunda (spec §5.1)
useSeoMeta({
  title: () => `${product.value?.name} | Çiçekçi`,
  description: () => product.value?.description || product.value?.name,
  ogTitle: () => product.value?.name,
  ogDescription: () => product.value?.description || product.value?.name,
  ogImage: () => product.value?.images?.[0]?.url_1200,
  ogUrl: () => `${cfg.siteUrl}/urun/${product.value?.slug}`,
  ogType: 'website',
})
</script>

<template>
  <div
    v-if="product"
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

    <div class="detay">
      <ProductGallery
        :images="product.images ?? []"
        :alt="product.name"
      />

      <div class="bilgi">
        <h1>{{ product.name }}</h1>

        <p class="fiyat">
          {{ formatPrice(product.price) }}
        </p>

        <div
          v-if="productCategories.length"
          class="etiketler"
        >
          <NuxtLink
            v-for="kategori in productCategories"
            :key="kategori.id"
            :to="`/kategori/${kategori.slug}`"
            class="etiket"
          >
            {{ kategori.name }}
          </NuxtLink>
        </div>

        <p
          v-if="product.description"
          class="aciklama"
        >
          {{ product.description }}
        </p>

        <WhatsAppButton :product="product" />

        <p class="not soluk">
          Siparişiniz WhatsApp üzerinden alınır. Mesajı gönderin, en kısa
          sürede dönüş yapalım.
        </p>
      </div>
    </div>
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

.detay {
  display: grid;
  gap: 1.5rem;
}

.bilgi h1 {
  margin-block-end: 0.25rem;
}

.fiyat {
  margin: 0 0 1rem;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--renk-vurgu);
}

.etiketler {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-block-end: 1.25rem;
}

.etiket {
  padding: 0.25rem 0.7rem;
  border: 1px solid var(--renk-cizgi);
  border-radius: 999px;
  background: var(--renk-zemin-alt);
  font-size: 0.85rem;
}

.etiket:hover {
  border-color: var(--renk-vurgu);
  color: var(--renk-vurgu);
}

.aciklama {
  margin-block-end: 1.5rem;
  white-space: pre-line;
}

.not {
  margin-block-start: 0.75rem;
  font-size: 0.85rem;
  text-align: center;
}

@media (min-width: 768px) {
  .detay {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 2.5rem;
    align-items: start;
  }
}
</style>
