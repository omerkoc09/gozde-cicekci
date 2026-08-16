<script setup lang="ts">
/**
 * Footer — referans 4 sütun: marka / kurumsal / yardım / sosyal + bülten.
 *
 * Bülten formu INERT (spec §2.1, §10) — backend'de karşılığı yok. Gönderince
 * "yakında" mesajı verir; sessizce yutup başarı numarası yapmaz.
 */
const { public: cfg } = useRuntimeConfig()
const yil = new Date().getFullYear()

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
            <li>
              <!--
                Esnafın panele girişi. Nuxt route'u değil, Caddy'nin admin
                SPA'ya yönlendirdiği ayrı bir yol — NuxtLink client-side
                gezinmeye çalışıp 404 vereceği için düz <a> kullanılıyor.
                nofollow: gizlilik değil (panel şifreli), gereksiz indeksleme
                engellensin diye.
              -->
              <a
                href="/idare"
                rel="nofollow"
                class="text-body-md text-on-surface-variant underline-offset-4 hover:text-primary hover:underline"
              >
                Yönetim
              </a>
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
              v-if="cfg.instagramUrl"
              :href="cfg.instagramUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="flex size-10 items-center justify-center rounded-full border border-outline-variant/50 text-on-surface-variant transition-colors hover:border-secondary hover:text-secondary"
              aria-label="Instagram"
            >
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true">
                <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.92-1.266.057-1.644.069-4.85.069-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838a6.162 6.162 0 1 0 0 12.324 6.162 6.162 0 0 0 0-12.324zm0 10.162a4 4 0 1 1 0-8 4 4 0 0 1 0 8zm6.406-11.845a1.44 1.44 0 1 0 0 2.881 1.44 1.44 0 0 0 0-2.881z" />
              </svg>
            </a>
          </div>
        </div>
      </div>

      <div class="mt-14 flex flex-col items-center gap-2 border-t border-outline-variant/30 pt-8 md:flex-row md:justify-between">
        <span class="flex items-center text-xs text-on-surface-variant">
          Made With
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" class="mx-1 text-error" aria-hidden="true">
            <path d="M12 21s-6.7-4.35-9.33-8.2C.87 10.1 1.4 6.6 4.2 4.9c2.1-1.28 4.7-.8 6.3 1.1L12 7.5l1.5-1.5c1.6-1.9 4.2-2.38 6.3-1.1 2.8 1.7 3.33 5.2 1.53 7.9C18.7 16.65 12 21 12 21z" />
          </svg>
          By <a href="https://linkedin.com/in/omerkoc-dev" target="_blank" rel="noopener noreferrer" class="ms-1 text-primary hover:underline">OmerKoc</a>
        </span>
        <p class="text-center text-xs text-on-surface-variant">
          © {{ yil }} Gözde Tasarım Çiçekçilik. Tüm hakları saklıdır.
        </p>
      </div>
    </div>
  </footer>
</template>
