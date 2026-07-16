<script setup lang="ts">
import type { Slide } from '~/types/api'

const props = withDefaults(defineProps<{
  slides: Slide[]
  /** Slayt başına bekleme (ms). */
  interval?: number
}>(), {
  interval: 4000,
})

const aktif = ref(0)

const cokluSlayt = computed(() => props.slides.length > 1)

// Oklar doğrudan fotoğrafın üstünde: yarı saydam açık zemin olmadan hem açık
// hem koyu görselde kayboluyorlar. DESIGN.md gölge sevmiyor — header'daki
// (bg-surface/80 + backdrop-blur) kalıbın aynısı kullanıldı.
const okClass = 'absolute top-1/2 grid size-11 -translate-y-1/2 place-items-center '
  + 'rounded-full bg-surface/75 text-primary backdrop-blur-sm transition-colors '
  + 'duration-200 hover:bg-surface/95 motion-reduce:transition-none'

const goTo = (i: number) => {
  // Mod ile sarmalıyoruz: son slayttan sonra başa, ilkten önce sona.
  aktif.value = (i + props.slides.length) % props.slides.length
}

const next = () => goTo(aktif.value + 1)
const prev = () => goTo(aktif.value - 1)

/**
 * Otomatik geçiş. setInterval yerine her adımda yeniden kurulan setTimeout:
 * kullanıcı butona bastığında sayaç sıfırdan başlasın — yoksa elle geçtiği
 * slayt yarım saniye sonra kayabilir.
 */
const duraklat = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

const clear = () => {
  if (timer) {
    clearTimeout(timer)
    timer = undefined
  }
}

// Erişilebilirlik: hareket azaltma tercihi açıksa otomatik geçiş hiç kurulmaz,
// kullanıcı slaytları elle gezer (WCAG 2.2.2 — otomatik hareket durdurulabilmeli).
// SSR'da matchMedia yok; sunucuda false, istemcide onMounted'da okunuyor.
const azHareket = ref(false)

// Sekme görünürlüğü reactive tutuluyor: dolum çubuğu da buna bakıyor,
// arka planda sayaç dururken çubuk dolmaya devam etmesin.
const sekmeGizli = ref(false)

/**
 * Otomatik geçiş açık mı — hem setTimeout hem dolum çubuğu bunu kullanır,
 * ikisi tek koşuldan beslendiği için çubuk hep gerçek sayacı gösterir.
 * duraklat burada YOK: duraklatınca sayaç durur ama çubuk donarak
 * kalan süreyi göstermeye devam eder (animationPlayState).
 */
const sayacCalisiyor = computed(() =>
  cokluSlayt.value && !azHareket.value && !sekmeGizli.value)

const kur = () => {
  clear()
  if (!sayacCalisiyor.value || duraklat.value)
    return

  timer = setTimeout(next, props.interval)
}

// aktif değişince sayaç yeniden kurulur — elle geçişte de tam süre beklenir.
watch([aktif, duraklat, sayacCalisiyor], kur)

// Sekme arkaplandayken saymanın anlamı yok: kullanıcı geri döndüğünde
// slaytların hızla akıp geçtiğini görmesin.
const gorunurlukDegisti = () => {
  sekmeGizli.value = document.hidden
}

// Sayaç yalnızca istemcide kurulur — SSR'da setTimeout beyhude.
onMounted(() => {
  const mq = window.matchMedia('(prefers-reduced-motion: reduce)')

  azHareket.value = mq.matches
  mq.addEventListener('change', e => {
    azHareket.value = e.matches
  })

  sekmeGizli.value = document.hidden
  document.addEventListener('visibilitychange', gorunurlukDegisti)

  kur()
})

onBeforeUnmount(() => {
  clear()
  if (import.meta.client)
    document.removeEventListener('visibilitychange', gorunurlukDegisti)
})
</script>

<template>
  <section
    class="relative isolate flex min-h-[520px] items-center overflow-hidden md:min-h-[620px]"
    role="region"
    aria-roledescription="carousel"
    aria-label="Öne çıkan koleksiyonlar"
    @mouseenter="duraklat = true"
    @mouseleave="duraklat = false"
    @focusin="duraklat = true"
    @focusout="duraklat = false"
  >
    <!-- Slaytlar üst üste; geçiş opacity ile. v-show değil v-for + opacity:
         hepsi DOM'da kalır, geçişte yeniden yükleme olmaz. -->
    <div
      v-for="(slide, i) in slides"
      :key="slide.id"
      class="absolute inset-0 -z-10 transition-opacity duration-700 ease-out motion-reduce:transition-none"
      :class="i === aktif ? 'opacity-100' : 'opacity-0'"
      :aria-hidden="i !== aktif"
    >
      <!-- İlk slayt LCP: eager + high priority. Diğerleri lazy —
           ziyaretçi çoğu zaman ilkini görüp kaydırıyor. -->
      <img
        :src="slide.url_1920"
        :srcset="`${slide.url_400} 400w, ${slide.url_1200} 1200w, ${slide.url_1920} 1920w`"
        sizes="100vw"
        alt=""
        aria-hidden="true"
        :fetchpriority="i === 0 ? 'high' : 'auto'"
        :loading="i === 0 ? 'eager' : 'lazy'"
        width="1920"
        height="1080"
        class="size-full object-cover"
      >
    </div>

    <div class="site-container">
      <!-- aria-live: slayt değişince ekran okuyucu yeni başlığı duyursun.
           polite — kullanıcının işini bölmez. -->
      <div
        class="max-w-xl"
        aria-live="polite"
        aria-atomic="true"
      >
        <template
          v-for="(slide, i) in slides"
          :key="slide.id"
        >
          <!-- hero-metin: fotoğrafın üstüne hiç katman koymuyoruz, o yüzden
               okunurluk gölgeyle sağlanıyor. DESIGN.md dekoratif gölgeyi
               reddediyor ama bu dekorasyon değil — açık renkli fotoğrafta
               beyaz metin gölgesiz kayboluyor (spec §9 kontrast). -->
          <div
            v-if="i === aktif"
            class="hero-metin"
          >
            <h1 class="font-serif text-4xl leading-tight tracking-tight text-white md:text-5xl lg:text-6xl lg:leading-[1.1]">
              {{ slide.title }}
            </h1>
            <p
              v-if="slide.subtitle"
              class="mt-5 max-w-md text-body-md text-white/95 md:text-body-lg"
            >
              {{ slide.subtitle }}
            </p>
            <NuxtLink
              to="/urunler"
              class="btn-primary text-label-caps mt-8"
            >
              Hemen Keşfet
            </NuxtLink>
          </div>
        </template>
      </div>
    </div>

    <!-- Tek slayt varsa kontroller gizli — basacak yer yok. -->
    <template v-if="cokluSlayt">
      <button
        type="button"
        :class="[okClass, 'left-4 md:left-6']"
        aria-label="Önceki slayt"
        @click="prev"
      >
        <Icon
          name="material-symbols:chevron-left"
          size="24"
        />
      </button>
      <button
        type="button"
        :class="[okClass, 'right-4 md:right-6']"
        aria-label="Sonraki slayt"
        @click="next"
      >
        <Icon
          name="material-symbols:chevron-right"
          size="24"
        />
      </button>

      <div class="absolute inset-x-0 bottom-6 flex justify-center gap-2.5">
        <button
          v-for="(slide, i) in slides"
          :key="slide.id"
          type="button"
          class="h-2 overflow-hidden rounded-full transition-all duration-300 motion-reduce:transition-none"
          :class="i === aktif
            ? 'w-8 bg-white/30'
            : 'w-2 bg-white/50 hover:bg-white/80'"
          :aria-label="`${i + 1}. slayta git`"
          :aria-current="i === aktif"
          @click="goTo(i)"
        >
          <!-- Dolum çubuğu: aktif noktanın içini süre boyunca doldurur, yani
               "sonraki slayta ne kadar kaldı"ı gösterir. key'de aktif var —
               slayt değişince eleman yeniden kurulur ve animasyon baştan başlar.
               Duraklatınca animation-play-state ile donar; sayaç da durduğu için
               çubuk ile gerçek kalan süre aynı kalır. -->
          <span
            v-if="i === aktif && sayacCalisiyor"
            :key="`ilerleme-${aktif}`"
            class="block h-full w-full origin-left rounded-full bg-white"
            :style="{
              animation: `hero-ilerleme ${interval}ms linear forwards`,
              animationPlayState: duraklat ? 'paused' : 'running',
            }"
          />
          <!-- Sayaç yokken (tek slayt, hareket azaltma, arka plan sekme)
               animasyon da yok: aktif nokta düz dolu görünür. -->
          <span
            v-else-if="i === aktif"
            class="block size-full rounded-full bg-white"
          />
        </button>
      </div>
    </template>
  </section>
</template>

<style scoped>
/* Fotoğrafın üstünde katman yok — okunurluk yalnızca bu gölgeye bağlı.
   Sert bir gölge değil: geniş ve yumuşak, metnin arkasında hafif bir
   koyuluk oluşturuyor, gölge gibi görünmüyor. İki katman: yakın olan
   harf kenarını keskinleştirir, geniş olan zemini bastırır. */
.hero-metin :is(h1, p) {
  text-shadow:
    0 1px 2px rgb(0 0 0 / 45%),
    0 2px 24px rgb(0 0 0 / 35%);
}
</style>

<style>
/* Dolum çubuğu: soldan sağa. transform kullanılıyor (width değil) —
   width animasyonu her karede layout hesaplatır, transform GPU'da kalır.
   scoped DEĞİL: animasyon inline style ile veriliyor, scoped attribute
   keyframes'e uygulanmadığı için isim global olmalı. */
@keyframes hero-ilerleme {
  from {
    transform: scaleX(0);
  }

  to {
    transform: scaleX(1);
  }
}
</style>

