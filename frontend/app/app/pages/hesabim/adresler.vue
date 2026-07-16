<script setup lang="ts">
import { MOCK_ADRESLER } from '~/utils/mockAccount'

definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Adreslerim | Gözde Tasarım Çiçekçilik' })

// Tüm adres işlemleri inert (spec §2.1) — backend'de adres yok.
const mesaj = ref('')

function yakinda() {
  mesaj.value = 'Adres yönetimi çok yakında açılıyor.'
}
</script>

<template>
  <div>
    <AccountHero
      title="Adresler"
      description="Teslimat ve fatura adreslerinizi yönetin, siparişlerinizi daha hızlı tamamlayın."
    />

    <div class="mt-8 grid gap-4 md:grid-cols-2">
      <div
        v-for="adres in MOCK_ADRESLER"
        :key="adres.id"
        class="rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-6"
      >
        <div class="flex items-start justify-between gap-4 border-b border-outline-variant/40 pb-4">
          <h2 class="text-label-caps text-on-surface">
            {{ adres.tur === 'fatura' ? 'Fatura Adresi' : 'Teslimat Adresi' }}
          </h2>
          <button
            type="button"
            class="rounded p-1 text-on-surface-variant transition-colors hover:text-primary"
            :aria-label="`${adres.tur === 'fatura' ? 'Fatura' : 'Teslimat'} adresini düzenle`"
            @click="yakinda"
          >
            <Icon name="material-symbols:edit-outline" size="18" />
          </button>
        </div>

        <address class="mt-5 space-y-1.5 not-italic text-body-md text-on-surface-variant">
          <p class="font-medium text-on-surface">
            {{ adres.ad }}
          </p>
          <p>{{ adres.satir }}</p>
          <p>{{ adres.ilce }}</p>
          <p>{{ adres.postaKodu }}</p>
          <p class="pt-2">
            {{ adres.telefon }}
          </p>
        </address>

        <button
          type="button"
          class="text-label-caps mt-6 inline-flex items-center gap-1.5 text-secondary hover:text-secondary-hover"
          @click="yakinda"
        >
          <Icon name="material-symbols:add" size="14" />
          Yeni {{ adres.tur === 'fatura' ? 'Fatura' : 'Teslimat' }} Adresi
        </button>
      </div>
    </div>

    <!-- Referanstaki "Send a Gift?" bölümü -->
    <div class="mt-10 border-t border-outline-variant/40 pt-10">
      <div class="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 class="font-serif text-2xl text-primary">
            Hediye mi göndereceksiniz?
          </h2>
          <p class="mt-2 max-w-md text-body-md text-on-surface-variant">
            Sevdiklerinizin adreslerini kaydedin, özel günlerde hızlı ve kolay
            teslimat yapalım.
          </p>
        </div>
        <button type="button" class="btn-primary text-label-caps shrink-0" @click="yakinda">
          Yeni Adres Ekle
        </button>
      </div>
    </div>

    <p v-if="mesaj" class="mt-5 text-center text-sm text-on-surface-variant" role="status">
      {{ mesaj }}
    </p>

    <AccountDemoNotice class="mt-8" />
  </div>
</template>
