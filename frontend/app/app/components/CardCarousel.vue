<script setup lang="ts">
/**
 * Yatay kaydırmalı kart şeridi. Kartlar alta inmez — yana kayar.
 *
 * Kaydırma tarayıcının kendi overflow'u ile yapılıyor, JS ile değil:
 * mobilde parmak, trackpad'de iki parmak, klavyede ok tuşları hepsi
 * kendiliğinden çalışır. JS yalnızca okları ve durumlarını yönetiyor.
 *
 * Kart genişliği burada değil, çağıranın slot içeriğinde belirlenir —
 * ürün kartı ile kategori kartının oranı farklı.
 */
withDefaults(defineProps<{
  /** Şeridin üstünde görünen başlık. */
  baslik: string
}>(), {})

const serit = ref<HTMLElement>()

// Oklar yalnızca gidilecek yer varsa aktif. Kaydırdıkça güncelleniyor.
const solaGidebilir = ref(false)
const sagaGidebilir = ref(false)

const durumGuncelle = () => {
  const el = serit.value
  if (!el)
    return

  // 1px tolerans: tarayıcılar kesirli piksel döndürebiliyor, tam uçta
  // ok aktif kalmasın.
  solaGidebilir.value = el.scrollLeft > 1
  sagaGidebilir.value = el.scrollLeft < el.scrollWidth - el.clientWidth - 1
}

// Bir "sayfa" = görünen alanın %80'i. Tam genişlik kaydırmak kartları
// tamamen değiştirir, kullanıcı nerede olduğunu kaybeder; %80 bağlam bırakır.
const kaydir = (yon: -1 | 1) => {
  const el = serit.value
  if (!el)
    return

  el.scrollBy({ left: yon * el.clientWidth * 0.8, behavior: 'smooth' })
}

// İçerik sonradan gelebilir (SSR → hydration, görsel yüklenmesi) ve pencere
// yeniden boyutlanabilir; iki durumda da okların doğruluğu bozulur.
let ro: ResizeObserver | undefined

onMounted(() => {
  durumGuncelle()

  if (serit.value) {
    ro = new ResizeObserver(durumGuncelle)
    ro.observe(serit.value)
  }
})

onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <section class="section-gap">
    <div class="site-container">
      <div class="flex items-baseline justify-between gap-4">
        <h2 class="font-serif text-3xl text-primary md:text-4xl">
          {{ baslik }}
        </h2>

        <!-- Sağdaki alan çağırana ait: "Tümünü Gör" gibi bağlantılar.
             Oklarla aynı satırda durur. -->
        <div class="flex items-center gap-4">
          <slot name="aksiyon" />

          <!-- Oklar yalnızca masaüstünde: mobilde parmakla kaydırmak zaten
               doğal, ok koymak yeri boşuna daraltır. Kaydırma gerekmiyorsa
               (kart azsa) ikisi de pasif olur. -->
          <div
            v-if="solaGidebilir || sagaGidebilir"
            class="hidden shrink-0 items-center gap-2 md:flex"
          >
            <button
              type="button"
              class="serit-ok"
              :disabled="!solaGidebilir"
              aria-label="Öncekiler"
              @click="kaydir(-1)"
            >
              <Icon
                name="material-symbols:chevron-left"
                size="20"
              />
            </button>
            <button
              type="button"
              class="serit-ok"
              :disabled="!sagaGidebilir"
              aria-label="Sonrakiler"
              @click="kaydir(1)"
            >
              <Icon
                name="material-symbols:chevron-right"
                size="20"
              />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Şerit site-container'ın DIŞINDA: kartlar ekran kenarına kadar
         kayabilsin, sağdaki kart kesilerek "devamı var" hissi versin.
         Hizayı içerideki padding sağlıyor (kenar boşluğu container ile aynı).

         snap-proximity, mandatory DEĞİL: mandatory ile tarayıcı açılışta ilk
         kartı hizaya oturtmak için şeridi padding kadar kaydırıyor
         (scrollLeft=64) ve kart soldan kesik başlıyor. proximity yalnızca
         kullanıcı yakınına geldiğinde yapışır — açılış hizası bozulmaz.
         scroll-pl: ok ile kaydırırken kart padding'in altında kalmasın. -->
    <div
      ref="serit"
      class="serit mt-10 flex snap-x snap-proximity gap-3 overflow-x-auto scroll-smooth px-5 scroll-pl-5 md:gap-6 md:px-8 md:scroll-pl-8 xl:px-16 xl:scroll-pl-16"
      @scroll.passive="durumGuncelle"
    >
      <slot />
    </div>
  </section>
</template>

<style scoped>
/* Kaydırma çubuğu gizli — kartların altında gri bir çizgi luxe tasarıma
   uymuyor. Kaydırma yine çalışıyor; masaüstünde oklar, mobilde parmak var. */
.serit {
  scrollbar-width: none;
}

.serit::-webkit-scrollbar {
  display: none;
}

.serit-ok {
  display: grid;
  place-items: center;
  block-size: 2.25rem;
  inline-size: 2.25rem;
  border: 1px solid var(--color-outline-variant);
  border-radius: 9999px;
  color: var(--color-primary);
  transition: border-color 0.2s, color 0.2s;
}

.serit-ok:hover:not(:disabled) {
  border-color: var(--color-secondary);
  color: var(--color-secondary);
}

.serit-ok:disabled {
  color: var(--color-outline-variant);
  cursor: default;
}
</style>
