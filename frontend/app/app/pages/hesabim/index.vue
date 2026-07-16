<script setup lang="ts">
import { MOCK_KULLANICI } from '~/utils/mockAccount'

definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Hesap Panom | Gözde Tasarım Çiçekçilik' })

// Referansta 4 kart vardı; "Siparişler" çıkarıldı (spec §6.2) — sipariş
// kavramı ve sayfası yok, tıklanınca gidecek yer olmazdı.
const KARTLAR = [
  {
    to: '/hesabim/adresler',
    ikon: 'material-symbols:location-on-outline',
    baslik: 'Adresler',
    metin: 'Teslimat ve fatura adreslerinizi düzenleyin.',
  },
  {
    to: '/hesabim/hesap-detaylari',
    ikon: 'material-symbols:person-outline',
    baslik: 'Hesap Detayları',
    metin: 'Kişisel bilgilerinizi ve şifrenizi güncelleyin.',
  },
  {
    to: '/hesabim/favoriler',
    ikon: 'material-symbols:favorite-outline',
    baslik: 'Favoriler',
    metin: 'Beğendiğiniz tasarımları listeleyin.',
  },
]
</script>

<template>
  <div>
    <AccountHero
      :title="`Merhaba ${MOCK_KULLANICI.ad}`"
      description="Hesap panonuzdan adreslerinizi yönetebilir, favori tasarımlarınızı listeleyebilir ve hesap detaylarınızı düzenleyebilirsiniz."
    />

    <div class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink
        v-for="kart in KARTLAR"
        :key="kart.to"
        :to="kart.to"
        class="rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-6 transition-colors hover:border-outline-variant"
      >
        <span class="flex size-11 items-center justify-center rounded-md bg-surface-container-low text-primary">
          <Icon :name="kart.ikon" size="20" />
        </span>
        <h2 class="mt-5 font-serif text-xl text-primary">
          {{ kart.baslik }}
        </h2>
        <p class="mt-2 text-body-md text-on-surface-variant">
          {{ kart.metin }}
        </p>
      </NuxtLink>
    </div>

    <AccountDemoNotice class="mt-10" />
  </div>
</template>
