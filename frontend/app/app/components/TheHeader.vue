<script setup lang="ts">
import type { Category } from '~/types/api'

/**
 * Sticky header — DESIGN.md §Elevation: backdrop blur, gölge yok, ince alt çizgi.
 *
 * Nav referanstaki 5'li yapıyı korur ama Türkçeleştirildi (spec §6.1) ve
 * "Özel Günler"/"Çiçekler" gerçek kategori eksenlerine bağlandı (spec §5):
 * occasion → Özel Günler, type → Çiçekler. İlk link ("Koleksiyonlar")
 * filtresiz /urunler'e gider, ayrı bir kategori ekseni değildir.
 *
 * Hesap ikonu artık gerçek: giriş durumuna göre /giris veya /hesabim'e gider
 * (üyelik spec'i 2026-08-13). Favori ikonu kaldırıldı — backend'de karşılığı
 * yok, mock favoriler ekranı silindi.
 * Sepet Faz 2'de gerçek oldu: rozet artık gerçek sayıyı gösterir.
 */
const emit = defineEmits<{ openCart: [] }>()

const { count: cartCount } = useCart()
const { me } = useCustomer()

const girisYapilmis = ref(false)
onMounted(async () => {
  girisYapilmis.value = !!(await me())
})

const { data: categories } = await useCategoryList()

const ozelGunler = computed(() =>
  (categories.value ?? []).filter((c: Category) => c.axis === 'occasion'),
)
const koleksiyonlar = computed(() =>
  (categories.value ?? []).filter((c: Category) => c.axis === 'type'),
)

const mobilAcik = ref(false)
const acikMenu = ref<'ozel' | 'koleksiyon' | null>(null)

const route = useRoute()
watch(() => route.fullPath, () => {
  mobilAcik.value = false
  acikMenu.value = null
})

// Mobil menü açıkken arka plan kaymasın
watch(mobilAcik, (acik) => {
  if (import.meta.client)
    document.body.style.overflow = acik ? 'hidden' : ''
})

onBeforeUnmount(() => {
  if (import.meta.client)
    document.body.style.overflow = ''
})

function menuAc(menu: 'ozel' | 'koleksiyon') {
  acikMenu.value = acikMenu.value === menu ? null : menu
}
</script>

<template>
  <header
    class="sticky top-0 z-50 border-b border-outline-variant/30 bg-surface/80 backdrop-blur-md"
    @keydown.escape="acikMenu = null"
  >
    <div class="flex w-full items-center gap-4 px-5 py-3 md:px-8 md:py-4 xl:px-16">
      <!-- Mobilde hamburger solda — logo'yu ortalamak için aksiyon grubuyla
           simetrik denge sağlıyor (bkz. aşağıdaki mobil hamburger butonu,
           burada da bir kopyası var ki logo gerçekten ortalansın). -->
      <button
        type="button"
        class="rounded p-2 text-primary transition-colors hover:text-secondary lg:hidden"
        :aria-expanded="mobilAcik"
        aria-label="Menü"
        @click="mobilAcik = !mobilAcik"
      >
        <Icon :name="mobilAcik ? 'material-symbols:close' : 'material-symbols:menu'" size="24" />
      </button>

      <!-- Marka bloğu: masaüstünde solda logo + arama yan yana; mobilde
           logo ortalanmış tek başına (arama hamburger menüsünde). -->
      <div class="flex flex-1 items-center justify-center gap-4 md:gap-6 lg:flex-none lg:justify-start">
        <TheLogo />
        <SearchBar class="hidden w-56 lg:flex xl:w-72" />
      </div>

      <!-- Masaüstü nav -->
      <nav
        class="ml-auto hidden items-center gap-8 lg:flex"
        aria-label="Ana menü"
      >
        <NuxtLink
          to="/urunler"
          class="text-nav-link text-on-surface-variant transition-colors duration-300 hover:text-secondary"
          active-class="border-b border-accent-gold font-semibold !text-primary"
        >
          Koleksiyonlar
        </NuxtLink>

        <HeaderNavDropdown
          v-if="ozelGunler.length"
          label="Özel Günler"
          :items="ozelGunler"
          :open="acikMenu === 'ozel'"
          @toggle="menuAc('ozel')"
          @close="acikMenu = null"
        />

        <HeaderNavDropdown
          v-if="koleksiyonlar.length"
          label="Çiçekler"
          :items="koleksiyonlar"
          :open="acikMenu === 'koleksiyon'"
          @toggle="menuAc('koleksiyon')"
          @close="acikMenu = null"
        />

        <NuxtLink
          to="/hakkimizda"
          class="text-nav-link text-on-surface-variant transition-colors duration-300 hover:text-secondary"
          active-class="border-b border-accent-gold font-semibold !text-primary"
        >
          Hakkımızda
        </NuxtLink>

        <NuxtLink
          to="/iletisim"
          class="text-nav-link text-on-surface-variant transition-colors duration-300 hover:text-secondary"
          active-class="border-b border-accent-gold font-semibold !text-primary"
        >
          İletişim
        </NuxtLink>
      </nav>

      <!-- Aksiyonlar -->
      <div class="ml-auto flex shrink-0 items-center gap-0.5 text-primary md:gap-2 lg:ml-0">
        <button
          type="button"
          class="relative rounded p-2 transition-colors hover:text-secondary"
          aria-label="Sepet"
          @click="emit('openCart')"
        >
          <Icon name="material-symbols:shopping-cart-outline" size="22" />
          <!-- Rozet artık gerçek: sepette ürün var. Redesign spec'i §2.1 rozeti
               "var olmayan içeriği iddia etmesin" diye kaldırmıştı — sepet gerçek
               olduğu için rozet artık yalan değil, bilgi. -->
          <span
            v-if="cartCount > 0"
            class="absolute -right-1 -top-1 flex size-4 items-center justify-center rounded-full bg-secondary text-[10px] font-medium text-white"
          >
            {{ cartCount > 9 ? '9+' : cartCount }}
          </span>
        </button>

        <NuxtLink
          :to="girisYapilmis ? '/hesabim' : '/giris'"
          class="hidden rounded p-2 transition-colors hover:text-secondary sm:block"
          :aria-label="girisYapilmis ? 'Hesabım' : 'Giriş Yap'"
        >
          <Icon name="material-symbols:person-outline" size="22" />
        </NuxtLink>
      </div>
    </div>

    <!-- Mobil menü -->
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      leave-active-class="transition duration-150 ease-in"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <nav
        v-if="mobilAcik"
        class="border-t border-outline-variant/30 bg-surface lg:hidden"
        aria-label="Mobil menü"
      >
        <div class="site-container max-h-[calc(100dvh-5rem)] overflow-y-auto py-4">
          <SearchBar class="mb-2 w-full border-b border-outline-variant/20 pb-4" />

          <NuxtLink to="/urunler" class="block border-b border-outline-variant/20 py-3 font-serif text-lg">
            Koleksiyonlar
          </NuxtLink>

          <div v-if="ozelGunler.length" class="border-b border-outline-variant/20 py-3">
            <p class="text-label-caps mb-2 text-secondary">
              Özel Günler
            </p>
            <NuxtLink
              v-for="k in ozelGunler"
              :key="k.id"
              :to="`/kategori/${k.slug}`"
              class="block py-1.5 text-body-md text-on-surface-variant"
            >
              {{ k.name }}
            </NuxtLink>
          </div>

          <div v-if="koleksiyonlar.length" class="border-b border-outline-variant/20 py-3">
            <p class="text-label-caps mb-2 text-secondary">
              Çiçekler
            </p>
            <NuxtLink
              v-for="k in koleksiyonlar"
              :key="k.id"
              :to="`/kategori/${k.slug}`"
              class="block py-1.5 text-body-md text-on-surface-variant"
            >
              {{ k.name }}
            </NuxtLink>
          </div>

          <NuxtLink to="/hakkimizda" class="block border-b border-outline-variant/20 py-3 font-serif text-lg">
            Hakkımızda
          </NuxtLink>
          <NuxtLink to="/iletisim" class="block py-3 font-serif text-lg">
            İletişim
          </NuxtLink>
        </div>
      </nav>
    </Transition>
  </header>
</template>
