<script setup lang="ts">
import type { NuxtError } from '#app'

/**
 * Hata sayfası — referansta yok (mockup'lar hep mutlu yolu gösteriyor) ama
 * gerçek sitede 404 kaçınılmaz: silinen ürün, yanlış yazılan URL, eski link
 * (spec §5.2). Jenerik bir hata ekranı premium hissi anında bozar.
 *
 * error.vue layout DIŞINDA render olur — header/footer'ı elle koyuyoruz.
 */
const props = defineProps<{ error: NuxtError }>()

const bulunamadi = computed(() => props.error?.statusCode === 404)

useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-background">
    <TheHeader />

    <main class="flex flex-1 items-center">
      <div class="site-container py-20 text-center">
        <img
          src="~/assets/img/gozde-icon.svg"
          alt=""
          aria-hidden="true"
          class="mx-auto h-20 w-auto opacity-60"
          width="68"
          height="100"
        >

        <p class="text-label-caps mt-8 text-secondary">
          {{ error?.statusCode ?? 500 }}
        </p>

        <h1 class="mt-3 font-serif text-3xl text-primary md:text-4xl">
          {{ bulunamadi ? 'Bu sayfa bulunamadı' : 'Bir şeyler ters gitti' }}
        </h1>

        <p class="mx-auto mt-4 max-w-md text-body-md text-on-surface-variant">
          {{ bulunamadi
            ? 'Aradığınız sayfa taşınmış veya kaldırılmış olabilir. Koleksiyonumuza göz atarak devam edebilirsiniz.'
            : 'Beklenmedik bir hata oluştu. Kısa süre içinde tekrar deneyebilir ya da bize WhatsApp’tan ulaşabilirsiniz.' }}
        </p>

        <div class="mt-10 flex flex-wrap justify-center gap-3">
          <NuxtLink to="/urunler" class="btn-primary text-label-caps" @click="clearError">
            Koleksiyonu Keşfet
          </NuxtLink>
          <NuxtLink to="/" class="btn-secondary text-label-caps" @click="clearError">
            Ana Sayfa
          </NuxtLink>
        </div>
      </div>
    </main>

    <TheFooter />
  </div>
</template>
