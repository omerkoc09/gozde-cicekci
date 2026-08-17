<script setup lang="ts">
import type { CartItemOption, Category } from '~/types/api'
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

/** İz yolunda gösterilecek kategori — varsa ilki. */
const izKategori = computed(() => productCategories.value[0])

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
    title: () => `${product.value?.name} | Gözde Tasarım Çiçekçilik`,
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

const { add } = useCart()
const sepetAcik = inject<Ref<boolean> | null>('sepetAcik', null)

const secimler = ref<CartItemOption[]>([])

/**
 * Her grubun İLK değeri otomatik seçili gelir — müşteri hiçbir şeye
 * dokunmasa bile sipariş "Ambalaj: Pembe" ile gider, esnaf ne
 * hazırlayacağını bilir. Zorunluluk kavramı bu yüzden kaldırıldı.
 *
 * immediate: SSR'da ve ürün değiştiğinde (aynı bileşen farklı slug'a
 * gezinirse) tekrar çalışır — bu projede "sayfa değişti ama onMounted
 * tekrar çalışmadı" hatası daha önce yaşandı, watch o tuzağı kapatıyor.
 */
watch(() => product.value?.option_groups, gruplar => {
  secimler.value = (gruplar ?? [])
    .filter(g => g.values.length > 0)
    .map(g => {
      const ilk = g.values[0]!

      return {
        value_id: ilk.id,
        group_name: g.name,
        value_name: ilk.name,
        swatch_hex: ilk.swatch_hex,
      }
    })
}, { immediate: true })

/** Takipsiz ürün in_stock=true geldiği için hiç tükendi görünmez. */
const tukendi = computed(() => product.value !== null && !product.value.in_stock)

function sepeteEkle() {
  // Tükenen ürün sepete eklenemez — buton zaten kilitli, bu ikinci kapı
  // (sunucu da rezervasyonda reddediyor, spec §4.1).
  if (!product.value || tukendi.value)
    return

  add({
    product_id: product.value.id,
    name: product.value.name,
    slug: product.value.slug,
    price: product.value.price,
    image: product.value.images?.[0]?.url_400 ?? '',
    quantity: 1,
    options: secimler.value,
  })

  if (sepetAcik)
    sepetAcik.value = true
}
</script>

<template>
  <div v-if="product">
    <div class="site-container py-10 md:py-16">
      <div class="grid gap-10 md:grid-cols-2 md:items-start md:gap-14 lg:gap-20">
        <ProductGallery
          :images="product.images ?? []"
          :alt="product.name"
        />

        <div>
          <BreadCrumb
            :items="[
              { label: 'Anasayfa', to: '/' },
              ...(izKategori ? [{ label: izKategori.name, to: `/kategori/${izKategori.slug}` }] : []),
              { label: product.name },
            ]"
          />

          <h1 class="mt-5 font-serif text-3xl leading-tight text-primary md:text-4xl lg:text-5xl">
            {{ product.name }}
          </h1>

          <p class="mt-4 text-body-lg text-on-surface">
            <!-- Eski fiyat üstü çizili, indirimli fiyat vurgulu (spec §6.1) -->
            <span
              v-if="product.old_price"
              class="me-2 text-base text-on-surface-variant/60 line-through"
            >
              {{ formatPrice(product.old_price) }}
            </span>
            <span :class="product.old_price ? 'font-medium text-primary' : ''">
              {{ formatPrice(product.price) }}
            </span>
            <span class="text-sm text-on-surface-variant/70">(KDV dahil)</span>
          </p>

          <p
            v-if="product.discount_remaining !== null"
            class="mt-1.5 text-sm text-primary"
          >
            Son {{ product.discount_remaining }} adet bu fiyata
          </p>

          <p
            v-if="tukendi"
            class="mt-3 inline-flex items-center gap-1.5 rounded bg-surface-container px-3 py-1.5 text-sm text-on-surface-variant"
          >
            <Icon name="material-symbols:info-outline" size="16" />
            Bu ürün şu anda tükendi
          </p>

          <hr class="my-7 border-outline-variant/40">

          <p v-if="product.description" class="whitespace-pre-line text-body-md text-on-surface-variant">
            {{ product.description }}
          </p>

          <!-- Özelleştirme: müşteri seçenek gruplarını burada seçer -->
          <ProductOptionSelector
            v-if="product.option_groups?.length"
            v-model="secimler"
            :groups="product.option_groups"
            class="my-6"
          />

          <!-- CTA'lar. "Sepete Ekle" gerçek (Faz 2); WhatsApp da hâlâ sitenin
               bir dönüşüm yolu, o yüzden hemen altında (spec §2.3). -->
          <div class="mt-8 space-y-3">
            <!--
              Buton hiç kilitlenmiyor: her grubun ilk değeri otomatik
              seçili geldiği için "önce seçim yap" durumu oluşmuyor.
            -->
            <button
              type="button"
              class="btn-primary text-label-caps w-full disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="tukendi"
              @click="sepeteEkle"
            >
              <Icon name="material-symbols:shopping-cart-outline" size="18" />
              {{ tukendi ? 'Tükendi' : 'Sepete Ekle' }}
            </button>

            <WhatsAppButton
              :product="product"
              :out-of-stock="tukendi"
            />
          </div>

          <p class="mt-3 text-center text-xs text-on-surface-variant">
            <template v-if="tukendi">
              Bu ürün tükendi. Ne zaman geleceğini WhatsApp'tan sorabilirsiniz.
            </template>
            <template v-else>
              Siparişiniz WhatsApp üzerinden alınır. Mesajı gönderin, en kısa
              sürede dönüş yapalım.
            </template>
          </p>

          <div v-if="productCategories.length" class="mt-8 flex flex-wrap gap-2">
            <NuxtLink
              v-for="kategori in productCategories"
              :key="kategori.id"
              :to="`/kategori/${kategori.slug}`"
              class="rounded-full border border-outline-variant/60 px-3.5 py-1.5 text-sm text-on-surface-variant transition-colors hover:border-primary hover:text-primary"
            >
              {{ kategori.name }}
            </NuxtLink>
          </div>

          <div class="mt-9">
            <AccordionItem title="Teslimat Bilgileri">
              <p>
                İstanbul içi siparişleriniz aynı gün teslim edilir. Teslimat
                saatini ve adres detaylarını WhatsApp üzerinden birlikte
                belirliyoruz.
              </p>
            </AccordionItem>
            <AccordionItem title="Tazelik Güvencesi">
              <p>
                Tüm aranjmanlarımız sipariş sonrası hazırlanır. Çiçekler
                mevsiminde ve en taze haliyle seçilir; görseldekiyle aynı
                kalitede teslim edilir.
              </p>
            </AccordionItem>
          </div>
        </div>
      </div>
    </div>

    <!-- Çiçek bakımı (referans bölümü) -->
    <section class="bg-surface-container-low">
      <div class="site-container py-16 md:py-20">
        <h2 class="text-center font-serif text-3xl text-primary md:text-4xl">
          Çiçek Bakımı
        </h2>
        <p class="mx-auto mt-3 max-w-lg text-center text-body-md text-on-surface-variant">
          Çiçeklerinizin ilk günkü tazeliğini daha uzun süre koruması için basit
          ama etkili ipuçları.
        </p>

        <div class="mt-12 grid gap-10 md:grid-cols-3 md:gap-8">
          <div
            v-for="ipucu in [
              { ikon: 'material-symbols:water-drop-outline', baslik: 'Suyu Tazeleyin', metin: 'Vazodaki suyu iki günde bir değiştirin. Suyun berrak ve temiz kalmasına özen gösterin.' },
              { ikon: 'material-symbols:content-cut', baslik: 'Sapları Kesin', metin: 'Her su değişiminde sapların ucundan 1-2 cm çapraz şekilde kesin. Bu, su emilimini artırır.' },
              { ikon: 'material-symbols:device-thermostat', baslik: 'Serin Tutun', metin: 'Çiçeklerinizi doğrudan güneş ışığından, klimalardan ve ısıtıcılardan uzak tutun.' },
            ]"
            :key="ipucu.baslik"
            class="text-center"
          >
            <div class="mx-auto flex size-12 items-center justify-center rounded-md border border-outline-variant/50 text-secondary">
              <Icon :name="ipucu.ikon" size="22" />
            </div>
            <h3 class="text-label-caps mt-4 text-on-surface">
              {{ ipucu.baslik }}
            </h3>
            <p class="mx-auto mt-2.5 max-w-xs text-body-md text-on-surface-variant">
              {{ ipucu.metin }}
            </p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
