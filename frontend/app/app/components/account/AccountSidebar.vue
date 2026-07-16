<script setup lang="ts">
/**
 * Hesap sidebar'ı — spec §6.2.
 *
 * Referans ekranlar tutarsızdı (kimi 4 kimi 5 item, İngilizce/Türkçe karışık,
 * "Wishlist" vs "Favoriler"). Tek yapıya sabitlendi ve Türkçeleştirildi.
 *
 * "Siparişler" BİLEREK yok: sipariş diye bir kavram yok, sayfası da yok —
 * tıklanınca gidecek yeri olmayan link, inert UI'ın bile kabul etmeyeceği
 * kadar kırık olurdu (kullanıcı onayıyla çıkarıldı).
 *
 * Mobilde yatay kaydırılabilir sekme olur (spec §8).
 */
const LINKLER = [
  { to: '/hesabim', label: 'Pano', ikon: 'material-symbols:dashboard-outline' },
  { to: '/hesabim/adresler', label: 'Adresler', ikon: 'material-symbols:location-on-outline' },
  { to: '/hesabim/hesap-detaylari', label: 'Hesap Detayları', ikon: 'material-symbols:person-outline' },
  { to: '/hesabim/favoriler', label: 'Favoriler', ikon: 'material-symbols:favorite-outline' },
]

// Çıkış inert — oturum diye bir şey yok (spec §2.1). Sahte "çıkış yapıldı"
// demek yerine durumu açıkça söylüyoruz.
const cikisMesaji = ref(false)
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
        class="text-nav-link flex w-full items-center gap-3 rounded px-4 py-3 text-error transition-colors hover:bg-error-container/40"
        @click="cikisMesaji = true"
      >
        <Icon name="material-symbols:logout" size="18" />
        Çıkış Yap
      </button>

      <p v-if="cikisMesaji" class="mt-2 px-4 text-xs text-on-surface-variant" role="status">
        Üyelik sistemi çok yakında açılıyor.
      </p>
    </div>
  </nav>
</template>
