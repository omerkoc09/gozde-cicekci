<script setup lang="ts">
const sepetAcik = ref(false)
provide('sepetAcik', sepetAcik)

// Hesap sayfaları giriş gerektirir, arama motorlarına gitmemeli.
useSeoMeta({ robots: 'noindex, nofollow' })

// Sidebar'daki "Merhaba, {ad}" için — her hesap sayfası zaten kendi
// onMounted'ında me() çağırıp giriş kontrolü yapıyor (yoksa /giris'e atıyor);
// burada sadece isim gösterimi için ayrı bir çağrı yapılıyor.
const { me } = useCustomer()
const musteriAdi = ref('')
onMounted(async () => {
  const musteri = await me()
  if (musteri)
    musteriAdi.value = musteri.name
})
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
              <p v-if="musteriAdi" class="mb-7 mt-1 text-body-md text-on-surface-variant">
                Merhaba, {{ musteriAdi }}
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
