<script setup lang="ts">
import { apiErrorMessage, apiErrorStatus } from '~/composables/useOrders'
import { donusYolunuCoz } from '~/utils/authRedirect'
import { telefonHatasi } from '~/utils/telefon'

useSeoMeta({
  title: 'Kayıt Ol | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})

const { register, me } = useCustomer()
const router = useRouter()
const route = useRoute()

// Kayıt sonrası dönülecek yol (giris.vue ile aynı mantık).
const donusYolu = computed(() => donusYolunuCoz(route.query.donus))

// Zaten giriş yapılmışsa tekrar kayıt formuna gerek yok.
onMounted(async () => {
  const musteri = await me()
  if (musteri)
    await router.replace(donusYolu.value)
})

const form = reactive({
  name: '',
  email: '',
  phone: '',
  password: '',
})

const gonderiliyor = ref(false)
const hata = ref('')

// Telefon hatası alan altında anlık gösterilir. Kullanıcı alana dokunana
// (blur) kadar gösterilmez — daha ilk harfte kırmızı uyarı çıkması rahatsız
// edici olurdu.
const telefonDokunuldu = ref(false)
const telefonHataMesaji = computed(() =>
  telefonDokunuldu.value ? telefonHatasi(form.phone) : '',
)

async function gonder() {
  hata.value = ''

  // İstemci tarafı ön kontrol: sunucuya gitmeden anında geri bildirim.
  // Asıl doğrulama backend'de (istemci kontrolü atlatılabilir).
  telefonDokunuldu.value = true
  const telHata = telefonHatasi(form.phone)
  if (telHata) {
    hata.value = telHata
    return
  }

  gonderiliyor.value = true

  try {
    await register({
      email: form.email,
      password: form.password,
      name: form.name,
      phone: form.phone,
    })
    await router.push(donusYolu.value)
  }
  catch (e) {
    // Backend kayıt hatalarında zaten açıklayıcı Türkçe mesaj dönüyor
    // (şifre kısa, e-posta geçersiz, hesap zaten var...). 409'da mesaj
    // beklenmedik şekilde boş gelirse kullanıcı ne yapacağını bilsin.
    hata.value = apiErrorStatus(e) === 409
      ? 'Bu e-posta ile zaten bir hesap var. Giriş yapmayı deneyin.'
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
        Kayıt Ol
      </h1>
      <p class="mt-3 text-body-md text-on-surface-variant">
        Hesap oluşturun, siparişlerinizi takip edin. Üyelik zorunlu değildir —
        üye olmadan da sipariş verebilirsiniz.
      </p>

      <form class="mt-10 space-y-5" @submit.prevent="gonder">
        <label class="block">
          <span class="text-label-caps text-secondary">Ad Soyad *</span>
          <input v-model="form.name" required autocomplete="name" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
        </label>

        <label class="block">
          <span class="text-label-caps text-secondary">E-posta *</span>
          <input v-model="form.email" required type="email" autocomplete="email" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
        </label>

        <label class="block">
          <span class="text-label-caps text-secondary">Telefon *</span>
          <input
            v-model="form.phone"
            required
            type="tel"
            inputmode="tel"
            autocomplete="tel"
            placeholder="0555 111 22 33"
            :aria-invalid="!!telefonHataMesaji"
            class="mt-1.5 w-full rounded border bg-surface px-3 py-2.5 text-body-md text-on-surface focus:outline-none"
            :class="telefonHataMesaji
              ? 'border-red-600 focus:border-red-600'
              : 'border-outline-variant/50 focus:border-secondary'"
            @blur="telefonDokunuldu = true"
          >
          <span v-if="telefonHataMesaji" class="mt-1 block text-body-sm text-red-700">
            {{ telefonHataMesaji }}
          </span>
        </label>

        <label class="block">
          <span class="text-label-caps text-secondary">Şifre *</span>
          <input v-model="form.password" required type="password" autocomplete="new-password" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
        </label>

        <p v-if="hata" class="text-body-md text-red-700" role="alert">
          {{ hata }}
        </p>

        <button
          type="submit"
          :disabled="gonderiliyor"
          class="btn-primary text-label-caps w-full disabled:opacity-60"
        >
          {{ gonderiliyor ? 'Kayıt oluşturuluyor...' : 'Kayıt Ol' }}
        </button>

        <p class="text-center text-body-md text-on-surface-variant">
          Zaten hesabınız var mı?
          <NuxtLink :to="{ path: '/giris', query: route.query.donus ? { donus: route.query.donus } : undefined }" class="text-secondary underline">
            Giriş yapın
          </NuxtLink>
        </p>
      </form>
    </div>
  </div>
</template>
