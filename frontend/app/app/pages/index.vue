<script setup lang="ts">
import heroGorsel from '~/assets/img/hero.webp'
import { kategoriGorseli } from '~/composables/useKategoriGorsel'

const { data: slides } = await useSlides()

// İki eksen ayrı çekiliyor: "Özel Günler" (occasion) ve "Çiçek Türlerine
// Göre" (type). Panelden hangi kategorinin öne çıkacağı seçiliyor; hiçbiri
// seçilmemişse ilgili bölüm hiç görünmez.
const { data: ozelGunler } = await useFeaturedCategories('occasion')
const { data: cicekTurleri } = await useFeaturedCategories('type')

// Tek şablon iki bölümü de basıyor — kart yapısı birebir aynı.
// Boş bölümler burada eleniyor: şablonda v-for + v-show olsaydı boş section
// DOM'da kalır ve section-gap boşluğu ortada asılı dururdu.
//
// eksen: "Tümünü Gör" /urunler?eksen=... ile gidiyor — CategoryFilter o
// eksen grubuna scroll ediyor. Spesifik kategori seçilmiyor, backend'de
// axis bazlı filtre yok (bkz. 2026-07-18 sipariş formu iyileştirmeleri kararı).
const kategoriBolumleri = computed(() =>
  [
    { baslik: 'Özel Günler', eksen: 'occasion' as const, kategoriler: ozelGunler.value ?? [] },
    { baslik: 'Çiçek Türleri', eksen: 'type' as const, kategoriler: cicekTurleri.value ?? [] },
  ].filter(b => b.kategoriler.length > 0))

// Ana sayfanın işi vitrin, katalog değil — panelden "öne çıkan" işaretlenen
// ürünler gelir (spec §5.2). Grid 4'lü olduğu için 8 = iki tam sıra.
const { data: products } = await useProductList({ oneCikan: true, limit: 8 })

useSeoMeta({
  title: 'Gözde Tasarım Çiçekçilik — Taze Çiçek ve Buket',
  description: 'Doğum günü, kutlama ve özel günler için özenle tasarlanmış taze çiçek aranjmanları. Sipariş WhatsApp üzerinden.',
  ogTitle: 'Gözde Tasarım Çiçekçilik — Taze Çiçek ve Buket',
  ogDescription: 'Duygularınızı çiçeklerle anlatın. Özenle tasarlanmış taze çiçek aranjmanları.',
  ogType: 'website',
})
</script>

<template>
  <div>
    <!-- Hero: panelden slayt girilmişse slider, girilmemişse statik görsel.
         Yedek şart — slider boşken ana sayfa başsız kalmasın. -->
    <HeroSlider
      v-if="slides?.length"
      :slides="slides"
    />

    <section
      v-else
      class="relative isolate flex min-h-[520px] items-center overflow-hidden md:min-h-[620px]"
    >
      <!-- Yerel asset, zaten webp'e optimize edilmiş (spec §4.2) — IPX'ten
           geçirmeye gerek yok, düz <img> yeterli ve daha hızlı. -->
      <img
        :src="heroGorsel"
        alt=""
        aria-hidden="true"
        fetchpriority="high"
        width="1408"
        height="768"
        class="absolute inset-0 -z-10 size-full object-cover"
      >
      <!-- Metin okunurluğu için: DESIGN.md gölge sevmiyor ama fotoğraf üstünde
           kontrast erişilebilirlik gereği (spec §9). -->
      <div
        class="absolute inset-0 -z-10 bg-gradient-to-r from-surface/85 via-surface/50 to-transparent"
        aria-hidden="true"
      />

      <div class="site-container">
        <div class="max-w-xl">
          <h1 class="font-serif text-4xl leading-tight tracking-tight text-primary md:text-5xl lg:text-6xl lg:leading-[1.1]">
            Duygularınızı<br>Çiçeklerle Anlatın
          </h1>
          <p class="mt-5 max-w-md text-body-md text-on-surface-variant md:text-body-lg">
            En özel anlarınızı taçlandıracak, özenle tasarlanmış taze çiçek
            aranjmanları. Sevdiklerinize zarafet hediye edin.
          </p>
          <NuxtLink to="/urunler" class="btn-primary text-label-caps mt-8">
            Hemen Keşfet
          </NuxtLink>
        </div>
      </div>
    </section>

    

    <!-- Öne çıkan kategoriler: Özel Günler (occasion) + Çiçek Türleri (type).
         Panelde o eksende hiçbir kategori öne çıkarılmamışsa bölüm görünmez.
         Kartlar yana kayar — 4'ten fazlası alta inmez.

         v-for yerine ayrı ayrı yazıldı: ikisi de doluysa aralarına
         GoldDivider koymak gerekiyor, tek bölüm varken divider anlamsız. -->
    <template v-for="(bolum, i) in kategoriBolumleri" :key="bolum.baslik">
      <div v-if="i > 0" class="site-container">
        <GoldDivider />
      </div>

      <CardCarousel :baslik="bolum.baslik">
        <template #aksiyon>
          <NuxtLink
            :to="`/urunler?eksen=${bolum.eksen}`"
            class="text-label-caps group flex shrink-0 items-center gap-1.5 text-secondary hover:text-secondary-hover"
          >
            Tümünü Gör
            <Icon
              name="material-symbols:arrow-forward"
              size="14"
              class="transition-transform group-hover:translate-x-0.5"
            />
          </NuxtLink>
        </template>

        <NuxtLink
          v-for="(kategori, j) in bolum.kategoriler"
          :key="kategori.id"
          :to="`/kategori/${kategori.slug}`"
          class="group relative block shrink-0 snap-start overflow-hidden rounded-lg
                 w-[66vw] sm:w-[45vw] lg:w-[calc((100%-3*1.5rem)/4)]"
          style="aspect-ratio: 3 / 4"
        >
          <!-- Panelden görsel yüklenmişse srcset ile küçük ekrana 400'lük
               gider; yedek görselde tek dosya var, srcset anlamsız. -->
          <img
            :src="kategoriGorseli(kategori, j)"
            :srcset="kategori.url_400 ? `${kategori.url_400} 400w, ${kategori.url_900} 900w` : undefined"
            sizes="(min-width: 1024px) 25vw, 66vw"
            alt=""
            aria-hidden="true"
            loading="lazy"
            width="900"
            height="1200"
            class="size-full object-cover transition-transform duration-700 group-hover:scale-105"
          >
          <span
            class="absolute inset-0 bg-gradient-to-t from-primary/70 via-primary/10 to-transparent"
            aria-hidden="true"
          />
          <h3 class="absolute inset-x-0 bottom-0 p-4 font-serif text-lg text-white md:p-5 md:text-xl">
            {{ kategori.name }}
          </h3>
        </NuxtLink>
      </CardCarousel>
    </template>

    <div class="site-container">
      <GoldDivider />
    </div>

    <!-- En çok tercih edilenler: panelden öne çıkarılan ürünler.
         Hiçbiri seçilmemişse bölüm hiç görünmez — "Ürünler yakında" demek
         yanıltıcı olurdu, katalogda ürün olabilir. -->
    <CardCarousel
      v-if="products?.length"
      baslik="En Çok Tercih Edilenler"
    >
      <template #aksiyon>
        <NuxtLink
          to="/urunler"
          class="text-label-caps group flex shrink-0 items-center gap-1.5 text-secondary hover:text-secondary-hover"
        >
          Tümünü Gör
          <Icon
            name="material-symbols:arrow-forward"
            size="14"
            class="transition-transform group-hover:translate-x-0.5"
          />
        </NuxtLink>
      </template>

      <ProductCard
        v-for="product in products"
        :key="product.id"
        :product="product"
        class="w-[66vw] shrink-0 snap-start sm:w-[45vw] lg:w-[calc((100%-3*1.5rem)/4)]"
      />
    </CardCarousel>

    <!-- Teslimat şeridi -->
    <section class="bg-surface-container-low">
      <div class="site-container py-14 text-center md:py-16">
        <Icon name="material-symbols:local-shipping-outline" size="30" class="text-secondary" />
        <h2 class="mt-3 font-serif text-xl text-primary md:text-2xl">
          Aynı Gün Teslimat
        </h2>
        <p class="mx-auto mt-3 max-w-md text-body-md text-on-surface-variant">
          İstanbul içi siparişlerinizde aynı gün, özenli ve güvenli teslimat
          hizmeti sunuyoruz. Sevdikleriniz beklemeye gelmez.
        </p>
      </div>
    </section>
  </div>
</template>
