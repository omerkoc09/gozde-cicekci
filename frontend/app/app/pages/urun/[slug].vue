<script setup lang="ts">
import type { Category } from '~/types/api'
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
// <script setup>'ta üst seviye return yasak; bunun yerine yönlendirme
// bayrağıyla altındaki setup'ı (kategori isteği + useSeoMeta) atlıyoruz ki
// eski slug'la gereksiz iş yapılmasın.
const yonlendiriliyor = product.value.slug !== slug

if (yonlendiriliyor) {
  await navigateTo(`/urun/${product.value.slug}`, {
    redirectCode: 301,
    replace: true,
  })
}

const { public: cfg } = useRuntimeConfig()

// Ürün yanıtında kategori İSİMLERİ yok, sadece category_ids — isimler için
// tüm kategori listesi çekiliyor (~16 kayıt, tek çağrı). Yönlendirmede atlanır.
const { data: categories } = yonlendiriliyor
  ? { data: ref<Category[]>([]) }
  : await useCategoryList()

const productCategories = computed(() =>
  (categories.value ?? []).filter(c => product.value?.category_ids?.includes(c.id)))

// Meta description: açıklama varsa ~160 karaktere kırp, yoksa isimle birebir
// aynı olmayan kısa bir genel metin üret (aksi halde başlık = açıklama).
const metaDescription = computed(() => {
  const p = product.value
  if (!p)
    return ''

  const aciklama = p.description?.trim()
  if (aciklama)
    return aciklama.length > 160 ? `${aciklama.slice(0, 157).trimEnd()}…` : aciklama

  return `${p.name} — taze çiçek ve buket. WhatsApp'tan sipariş verin.`
})

// WhatsApp önizlemesinin çalıştığı yer — og:image SSR'da gelmek zorunda (spec §5.1)
if (!yonlendiriliyor) {
  useSeoMeta({
    title: () => `${product.value?.name} | Çiçekçi`,
    // Açıklama uzun olabilir; meta description ~160 karakterle sınırlanıyor.
    // Boşsa isim değil kısa bir genel metin — başlıkla birebir aynı olmasın.
    description: () => metaDescription.value,
    ogTitle: () => product.value?.name,
    ogDescription: () => metaDescription.value,
    ogImage: () => product.value?.images?.[0]?.url_1200,
    ogUrl: () => `${cfg.siteUrl}/urun/${product.value?.slug}`,
    ogType: 'website',
  })
}
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
