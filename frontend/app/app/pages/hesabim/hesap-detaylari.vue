<script setup lang="ts">
import { MOCK_KULLANICI } from '~/utils/mockAccount'

definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Hesap Detayları | Gözde Tasarım Çiçekçilik' })

// Form inert (spec §2.1) — kaydedecek backend yok. Alanlar dolu görünür ama
// gönderince sahte "kaydedildi" demez.
const mesaj = ref('')

function kaydet() {
  mesaj.value = 'Hesap düzenleme çok yakında açılıyor.'
}
</script>

<template>
  <div>
    <AccountHero
      title="Hesap Detayları"
      description="Kişisel bilgilerinizi görüntüleyin ve güncelleyin."
    />

    <form class="mt-8 rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-6 md:p-8" @submit.prevent="kaydet">
      <div class="grid gap-6 md:grid-cols-2">
        <div>
          <label for="ad" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Ad Soyad
          </label>
          <input
            id="ad"
            :value="MOCK_KULLANICI.tamAd"
            type="text"
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>

        <div>
          <label for="eposta" class="text-label-caps mb-2 block text-on-surface-variant/70">
            E-posta
          </label>
          <input
            id="eposta"
            :value="MOCK_KULLANICI.eposta"
            type="email"
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>

        <div>
          <label for="telefon" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Telefon
          </label>
          <input
            id="telefon"
            :value="MOCK_KULLANICI.telefon"
            type="tel"
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>

        <div>
          <label for="sifre" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Şifre
          </label>
          <input
            id="sifre"
            value="••••••••"
            type="password"
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>
      </div>

      <div class="mt-9 flex flex-wrap items-center gap-4">
        <button type="submit" class="btn-primary text-label-caps">
          Değişiklikleri Kaydet
        </button>
        <p v-if="mesaj" class="text-sm text-on-surface-variant" role="status">
          {{ mesaj }}
        </p>
      </div>
    </form>

    <!-- Hesap işlemleri (referans bölümü) -->
    <div class="mt-10 border-t border-outline-variant/40 pt-8">
      <h2 class="text-label-caps text-on-surface-variant/70">
        Hesap İşlemleri
      </h2>
      <div class="mt-5 flex flex-wrap items-center gap-4">
        <button type="button" class="btn-secondary text-label-caps" @click="kaydet">
          <Icon name="material-symbols:logout" size="16" />
          Çıkış Yap
        </button>
        <button
          type="button"
          class="text-label-caps inline-flex items-center gap-2 px-2 text-error hover:underline"
          @click="kaydet"
        >
          <Icon name="material-symbols:delete-forever-outline" size="16" />
          Hesabı Sil
        </button>
      </div>
    </div>

    <AccountDemoNotice class="mt-8" />
  </div>
</template>
