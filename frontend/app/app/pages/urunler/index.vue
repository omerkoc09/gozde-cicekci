<script setup lang="ts">
const route = useRoute()

// Filtre state'i URL'de (spec §5.6) — computed olarak okunuyor, useProductList
// reactive aldığı için query değişince liste kendiliğinden yenileniyor.
const amac = computed(() => route.query.amac as string | undefined)
const tip = computed(() => route.query.tip as string | undefined)

const { data: products, status } = await useProductList({
  amac,
  tip,
  limit: 100,
})

const filtreVar = computed(() => !!amac.value || !!tip.value)

// Referanstaki "Daha Fazla Göster" — backend sayfalama vermiyor, liste zaten
// tek seferde geliyor (limit 100). Sunucuyu değiştirmeden aynı deneyimi
// vermek için istemci tarafında kademeli gösteriyoruz.
const ADIM = 12
const gosterilen = ref(ADIM)

watch([amac, tip], () => { gosterilen.value = ADIM })

const gorunenUrunler = computed(() => (products.value ?? []).slice(0, gosterilen.value))
const dahaVar = computed(() => (products.value?.length ?? 0) > gosterilen.value)

// Filtre kombinasyonları noindex (spec §4.2): 10 occasion × 6 type = 60
// kombinasyon Google'da ince içerik üretir. Tekil kategoriler kendi
// path'lerinde indexleniyor (/kategori/[slug]).
useSeoMeta({
  title: 'Çiçek Koleksiyonumuz | Gözde Tasarım Çiçekçilik',
  description: 'Mevsimin en taze çiçekleriyle hazırlanan, her biri benzersiz bir hikaye anlatan özel tasarımlarımızı keşfedin.',
  robots: () => (filtreVar.value ? 'noindex, follow' : 'index, follow'),
})
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <!-- Başlık + filtre yan yana (referans düzeni) -->
    <div class="flex flex-col gap-8 lg:flex-row lg:items-start lg:justify-between lg:gap-16">
      <div class="max-w-lg">
        <h1 class="font-serif text-4xl text-primary md:text-5xl">
          Çiçek Koleksiyonumuz
        </h1>
        <p class="mt-4 text-body-md text-on-surface-variant">
          Mevsimin en taze çiçekleriyle hazırlanan, her biri benzersiz bir hikaye
          anlatan özel tasarımlarımızı keşfedin.
        </p>
      </div>

      <div class="lg:max-w-md lg:pt-2">
        <CategoryFilter />
      </div>
    </div>

    <!-- Liste -->
    <div v-if="status === 'pending'" class="mt-14 grid grid-cols-2 gap-3 md:gap-6 lg:grid-cols-4">
      <ProductCardSkeleton v-for="i in 8" :key="i" />
    </div>

    <div v-else-if="gorunenUrunler.length" class="mt-14">
      <div class="grid grid-cols-2 gap-3 md:gap-6 lg:grid-cols-4">
        <ProductCard
          v-for="product in gorunenUrunler"
          :key="product.id"
          :product="product"
        />
      </div>

      <div v-if="dahaVar" class="mt-16 flex justify-center">
        <button type="button" class="btn-secondary text-label-caps" @click="gosterilen += ADIM">
          Daha Fazla Göster
        </button>
      </div>
    </div>

    <EmptyState
      v-else
      icon="material-symbols:search-off"
      title="Bu filtreye uyan ürün yok"
      description="Farklı bir kategori deneyebilir ya da tüm koleksiyonu görüntüleyebilirsiniz."
    >
      <NuxtLink v-if="filtreVar" to="/urunler" class="btn-secondary text-label-caps mt-7">
        Filtreyi Temizle
      </NuxtLink>
    </EmptyState>
  </div>
</template>
