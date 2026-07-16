<script setup lang="ts">
/**
 * Footer — referans 4 sütun: marka / kurumsal / yardım / sosyal + bülten.
 *
 * Bülten formu INERT (spec §2.1, §10) — backend'de karşılığı yok. Gönderince
 * "yakında" mesajı verir; sessizce yutup başarı numarası yapmaz.
 */
const { public: cfg } = useRuntimeConfig()
const yil = new Date().getFullYear()

const eposta = ref('')
const bultenMesaji = ref('')

function bultenGonder() {
  bultenMesaji.value = 'Bülten kaydı çok yakında açılıyor.'
  eposta.value = ''
}
</script>

<template>
  <footer class="mt-auto border-t border-outline-variant/30 bg-surface-container-low">
    <div class="site-container py-16 md:py-20">
      <div class="grid grid-cols-1 gap-10 md:grid-cols-2 lg:grid-cols-4 lg:gap-8">
        <!-- Marka -->
        <div>
          <p class="font-serif text-2xl leading-tight text-primary">
            Gözde Tasarım<br>Çiçekçilik
          </p>
          <p class="mt-4 max-w-xs text-body-md text-on-surface-variant">
            Sanatı doğayla buluşturan zarif tasarımlar. Duygularınızı en güzel
            çiçeklerle ifade edin.
          </p>
        </div>

        <!-- Kurumsal -->
        <div>
          <p class="text-label-caps mb-4 text-secondary">
            Kurumsal
          </p>
          <ul class="space-y-2.5">
            <li>
              <NuxtLink to="/hakkimizda" class="text-body-md text-on-surface-variant underline-offset-4 hover:text-primary hover:underline">
                Hakkımızda
              </NuxtLink>
            </li>
            <li>
              <NuxtLink to="/iletisim" class="text-body-md text-on-surface-variant underline-offset-4 hover:text-primary hover:underline">
                İletişim
              </NuxtLink>
            </li>
            <li>
              <NuxtLink to="/urunler" class="text-body-md text-on-surface-variant underline-offset-4 hover:text-primary hover:underline">
                Çiçekler
              </NuxtLink>
            </li>
          </ul>
        </div>

        <!-- Yardım -->
        <div>
          <p class="text-label-caps mb-4 text-secondary">
            Yardım
          </p>
          <ul class="space-y-2.5">
            <li v-if="cfg.contactPhone">
              <a
                :href="`tel:${cfg.contactPhone.replace(/\s/g, '')}`"
                class="text-body-md text-on-surface-variant underline-offset-4 hover:text-primary hover:underline"
              >
                {{ cfg.contactPhone }}
              </a>
            </li>
            <li v-if="cfg.contactHours" class="text-body-md text-on-surface-variant">
              {{ cfg.contactHours }}
            </li>
            <li v-if="cfg.contactAddress" class="max-w-xs text-body-md text-on-surface-variant">
              {{ cfg.contactAddress }}
            </li>
          </ul>
        </div>

        <!-- Sosyal + bülten -->
        <div>
          <p class="text-label-caps mb-4 text-secondary">
            Bizi Takip Edin
          </p>
          <div class="flex gap-3">
            <a
              href="https://instagram.com"
              target="_blank"
              rel="noopener noreferrer"
              class="flex size-10 items-center justify-center rounded-full border border-outline-variant/50 text-on-surface-variant transition-colors hover:border-secondary hover:text-secondary"
              aria-label="Instagram"
            >
              <Icon name="material-symbols:photo-camera-outline" size="18" />
            </a>
            <a
              href="https://facebook.com"
              target="_blank"
              rel="noopener noreferrer"
              class="flex size-10 items-center justify-center rounded-full border border-outline-variant/50 text-on-surface-variant transition-colors hover:border-secondary hover:text-secondary"
              aria-label="Facebook"
            >
              <Icon name="material-symbols:thumb-up-outline" size="18" />
            </a>
          </div>

          <p class="text-label-caps mb-3 mt-8 text-secondary">
            Bültene Kayıt Olun
          </p>
          <form class="flex items-center gap-2 border-b border-outline-variant pb-1" @submit.prevent="bultenGonder">
            <label for="footer-eposta" class="sr-only">E-posta adresiniz</label>
            <input
              id="footer-eposta"
              v-model="eposta"
              type="email"
              required
              placeholder="E-posta adresiniz"
              class="min-w-0 flex-1 bg-transparent py-1.5 text-body-md text-on-surface placeholder:text-on-surface-variant/60 focus:outline-none"
            >
            <button type="submit" class="text-label-caps shrink-0 text-secondary hover:text-secondary-hover">
              Gönder
            </button>
          </form>
          <p v-if="bultenMesaji" class="mt-2 text-xs text-on-surface-variant" role="status">
            {{ bultenMesaji }}
          </p>
        </div>
      </div>

      <div class="mt-14 border-t border-outline-variant/30 pt-8">
        <p class="text-center text-xs text-on-surface-variant">
          © {{ yil }} Gözde Tasarım Çiçekçilik. Tüm hakları saklıdır.
        </p>
      </div>
    </div>
  </footer>
</template>
