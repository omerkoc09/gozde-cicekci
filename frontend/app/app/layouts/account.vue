<script setup lang="ts">
import { MOCK_KULLANICI } from '~/utils/mockAccount'

const sepetAcik = ref(false)
provide('sepetAcik', sepetAcik)

// Hesap sayfaları demo (spec §2.1) — arama motorlarına gitmemeli.
useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-background">
    <TheHeader @open-cart="sepetAcik = true" />

    <main class="flex-1 pb-16 lg:pb-0">
      <div class="site-container py-10 md:py-14">
        <div class="grid gap-8 md:grid-cols-[220px_minmax(0,1fr)] md:gap-12 lg:grid-cols-[240px_minmax(0,1fr)]">
          <!-- min-w-0: grid hücresi varsayılan olarak içeriğine göre büyür;
               bu olmadan sidebar'ın yatay kaydırma şeridi hücreyi şişirip
               sayfayı taşırıyor. -->
          <aside class="min-w-0">
            <div class="hidden md:block">
              <p class="font-serif text-2xl text-primary">
                Hesabım
              </p>
              <p class="mb-7 mt-1 text-body-md text-on-surface-variant">
                Merhaba, {{ MOCK_KULLANICI.ad }}
              </p>
            </div>

            <AccountSidebar />
          </aside>

          <div>
            <slot />
          </div>
        </div>
      </div>
    </main>

    <TheFooter />

    <TheCartDrawer v-model="sepetAcik" />
    <WhatsAppFab />
    <TheBottomNav @open-cart="sepetAcik = true" />
  </div>
</template>
