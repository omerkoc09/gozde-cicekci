<script setup lang="ts">
import { apiErrorMessage, apiErrorStatus } from '~/composables/useOrders'
import { donusYolunuCoz } from '~/utils/authRedirect'

useSeoMeta({
  title: 'Giriş Yap | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})

const { login, me } = useCustomer()
const router = useRouter()
const route = useRoute()

// Giriş sonrası nereye dönüleceği (?donus=/siparis ile checkout'tan gelinir).
const donusYolu = computed(() => donusYolunuCoz(route.query.donus))

// Zaten giriş yapılmışsa tekrar giriş formuna gerek yok.
onMounted(async () => {
  const musteri = await me()
  if (musteri)
    await router.replace(donusYolu.value)
})

const form = reactive({
  email: '',
  password: '',
})

const gonderiliyor = ref(false)
const hata = ref('')

async function gonder() {
  hata.value = ''
  gonderiliyor.value = true

  try {
    await login({ email: form.email, password: form.password })
    await router.push(donusYolu.value)
  }
  catch (e) {
    // Giriş ucunda 401 tek bir şey demek: e-posta ya da şifre yanlış.
    // Backend'in sabit "Yetkisiz" mesajı kullanıcıya ne yapacağını
    // söylemiyor — burada duruma özel karşılığını veriyoruz. (Hangisinin
    // yanlış olduğunu SÖYLEMİYORUZ: e-posta kayıtlı mı sızdırmamak için
    // backend de ikisine aynı hatayı dönüyor.)
    hata.value = apiErrorStatus(e) === 401
      ? 'E-posta veya şifre hatalı. Lütfen kontrol edip tekrar deneyin.'
      : apiErrorMessage(e)
  }
  finally {
    gonderiliyor.value = false
  }
}
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <div class="mx-auto max-w-md">
      <h1 class="font-serif text-3xl text-primary md:text-4xl">
        Giriş Yap
      </h1>
      <p class="mt-3 text-body-md text-on-surface-variant">
        Hesabınıza giriş yapın, sipariş geçmişinizi görüntüleyin.
      </p>

      <form class="mt-10 space-y-5" @submit.prevent="gonder">
        <label class="block">
          <span class="text-label-caps text-secondary">E-posta *</span>
          <input v-model="form.email" required type="email" autocomplete="email" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
        </label>

        <label class="block">
          <span class="text-label-caps text-secondary">Şifre *</span>
          <input v-model="form.password" required type="password" autocomplete="current-password" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
        </label>

        <p v-if="hata" class="text-body-md text-red-700" role="alert">
          {{ hata }}
        </p>

        <button
          type="submit"
          :disabled="gonderiliyor"
          class="btn-primary text-label-caps w-full disabled:opacity-60"
        >
          {{ gonderiliyor ? 'Giriş yapılıyor...' : 'Giriş Yap' }}
        </button>

        <p class="text-center text-body-md text-on-surface-variant">
          Hesabınız yok mu?
          <NuxtLink :to="{ path: '/kayit', query: route.query.donus ? { donus: route.query.donus } : undefined }" class="text-secondary underline">
            Kayıt olun
          </NuxtLink>
        </p>
      </form>
    </div>
  </div>
</template>
