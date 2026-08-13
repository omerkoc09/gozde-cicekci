<script setup lang="ts">
/**
 * Hesap sidebar'ı — üyelik/müşteri hesabı spec'i (2026-08-13).
 *
 * Favoriler ve Adresler kaldırıldı: backend'de karşılıkları yok, mock
 * ekranları silindi. Kalan 2 item gerçek: Pano (sipariş geçmişi), Hesap
 * Detayları (profil + şifre) — ikisi de gerçek /customer/* uçlarına bağlı.
 *
 * Mobilde yatay kaydırılabilir sekme olur (spec §8).
 */
const LINKLER = [
  { to: '/hesabim', label: 'Pano', ikon: 'material-symbols:dashboard-outline' },
  { to: '/hesabim/hesap-detaylari', label: 'Hesap Detayları', ikon: 'material-symbols:person-outline' },
]

const { logout } = useCustomer()
const router = useRouter()
const cikisYapiliyor = ref(false)

async function cikisYap() {
  cikisYapiliyor.value = true
  try {
    await logout()
  }
  finally {
    cikisYapiliyor.value = false
    await router.push('/giris')
  }
}
</script>

<template>
  <nav aria-label="Hesap menüsü">
    <!-- Mobil: yatay sekme. Kenardan kenara kaysın diye negatif margin ile
         site-container padding'i dengeleniyor. -->
    <div class="-mx-5 flex gap-1 overflow-x-auto px-5 pb-1 [scrollbar-width:none] md:hidden">
      <NuxtLink
        v-for="l in LINKLER"
        :key="l.to"
        :to="l.to"
        class="text-nav-link flex shrink-0 items-center gap-2 rounded px-3.5 py-2.5 text-on-surface-variant"
        active-class="bg-primary !text-on-primary"
      >
        <Icon :name="l.ikon" size="17" />
        {{ l.label }}
      </NuxtLink>
    </div>

    <!-- Masaüstü: dikey liste -->
    <div class="hidden md:block">
      <NuxtLink
        v-for="l in LINKLER"
        :key="l.to"
        :to="l.to"
        class="text-nav-link flex items-center gap-3 rounded px-4 py-3 text-on-surface-variant transition-colors hover:bg-surface-container-low hover:text-primary"
        active-class="bg-primary !text-on-primary hover:!bg-primary"
      >
        <Icon :name="l.ikon" size="18" />
        {{ l.label }}
      </NuxtLink>

      <hr class="my-4 border-outline-variant/40">

      <button
        type="button"
        :disabled="cikisYapiliyor"
        class="text-nav-link flex w-full items-center gap-3 rounded px-4 py-3 text-error transition-colors hover:bg-error-container/40 disabled:opacity-60"
        @click="cikisYap"
      >
        <Icon name="material-symbols:logout" size="18" />
        {{ cikisYapiliyor ? 'Çıkış yapılıyor...' : 'Çıkış Yap' }}
      </button>
    </div>
  </nav>
</template>
