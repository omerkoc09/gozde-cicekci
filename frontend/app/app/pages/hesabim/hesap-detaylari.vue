<script setup lang="ts">
import type { Customer } from '~/types/api'
import { apiErrorMessage, apiErrorStatus } from '~/composables/useOrders'
import { telefonHatasi } from '~/utils/telefon'

definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Hesap Detayları | Gözde Tasarım Çiçekçilik' })

const { me, updateProfile, logout } = useCustomer()
const router = useRouter()

const musteri = ref<Customer | null>(null)
const yukleniyor = ref(true)

const profil = reactive({ name: '', phone: '' })
const profilMesaj = ref('')
const profilHata = ref('')
const profilKaydediliyor = ref(false)

const sifreForm = reactive({ current_password: '', new_password: '' })
const sifreMesaj = ref('')
const sifreHata = ref('')
const sifreKaydediliyor = ref(false)

onMounted(async () => {
  const sonuc = await me()
  if (!sonuc) {
    await router.replace('/giris')
    return
  }
  musteri.value = sonuc
  profil.name = sonuc.name
  profil.phone = sonuc.phone
  yukleniyor.value = false
})

async function profilKaydet() {
  profilMesaj.value = ''
  profilHata.value = ''

  // Kayıt formundaki telefon kuralı burada da geçerli — asıl doğrulama
  // backend'de, bu yalnızca anında geri bildirim.
  const telHata = telefonHatasi(profil.phone)
  if (telHata) {
    profilHata.value = telHata
    return
  }

  profilKaydediliyor.value = true

  try {
    const guncel = await updateProfile({ name: profil.name, phone: profil.phone })
    musteri.value = guncel
    profilMesaj.value = 'Bilgileriniz güncellendi.'
  }
  catch (e) {
    profilHata.value = apiErrorMessage(e)
  }
  finally {
    profilKaydediliyor.value = false
  }
}

async function sifreKaydet() {
  sifreMesaj.value = ''
  sifreHata.value = ''
  sifreKaydediliyor.value = true

  try {
    await updateProfile({
      name: profil.name,
      phone: profil.phone,
      current_password: sifreForm.current_password,
      new_password: sifreForm.new_password,
    })
    sifreMesaj.value = 'Şifreniz güncellendi.'
    sifreForm.current_password = ''
    sifreForm.new_password = ''
  }
  catch (e) {
    // Şifre değiştirme ucunda 401 = mevcut şifre yanlış (oturum zaten
    // geçerli, aksi halde sayfa /giris'e yönlenirdi). Sabit "Yetkisiz"
    // yerine kullanıcıya hangi alanı düzelteceğini söylüyoruz.
    sifreHata.value = apiErrorStatus(e) === 401
      ? 'Mevcut şifreniz hatalı.'
      : apiErrorMessage(e)
  }
  finally {
    sifreKaydediliyor.value = false
  }
}

const cikisYapiliyor = ref(false)
async function cikisYap() {
  cikisYapiliyor.value = true
  try {
    await logout()
  }
  finally {
    cikisYapiliyor.value = false
    await router.push('/giris')
  }
}
</script>

<template>
  <div>
    <AccountHero
      title="Hesap Detayları"
      description="Kişisel bilgilerinizi görüntüleyin ve güncelleyin."
    />

    <form v-if="musteri" class="mt-8 rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-6 md:p-8" @submit.prevent="profilKaydet">
      <div class="grid gap-6 md:grid-cols-2">
        <div>
          <label for="ad" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Ad Soyad
          </label>
          <input
            id="ad"
            v-model="profil.name"
            type="text"
            required
            maxlength="120"
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>

        <div>
          <label for="eposta" class="text-label-caps mb-2 block text-on-surface-variant/70">
            E-posta
          </label>
          <input
            id="eposta"
            :value="musteri.email"
            type="email"
            disabled
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface-variant/70"
          >
        </div>

        <div>
          <label for="telefon" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Telefon
          </label>
          <input
            id="telefon"
            v-model="profil.phone"
            type="tel"
            required
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>
      </div>

      <div class="mt-9 flex flex-wrap items-center gap-4">
        <button type="submit" :disabled="profilKaydediliyor" class="btn-primary text-label-caps disabled:opacity-60">
          {{ profilKaydediliyor ? 'Kaydediliyor...' : 'Değişiklikleri Kaydet' }}
        </button>
        <p v-if="profilMesaj" class="text-sm text-on-surface-variant" role="status">
          {{ profilMesaj }}
        </p>
        <p v-if="profilHata" class="text-sm text-red-700" role="alert">
          {{ profilHata }}
        </p>
      </div>
    </form>

    <!-- Şifre değiştir -->
    <form v-if="musteri" class="mt-8 rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-6 md:p-8" @submit.prevent="sifreKaydet">
      <h2 class="text-label-caps text-on-surface-variant/70">
        Şifre Değiştir
      </h2>

      <div class="mt-5 grid gap-6 md:grid-cols-2">
        <div>
          <label for="mevcut-sifre" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Mevcut Şifre
          </label>
          <input
            id="mevcut-sifre"
            v-model="sifreForm.current_password"
            type="password"
            autocomplete="current-password"
            required
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>

        <div>
          <label for="yeni-sifre" class="text-label-caps mb-2 block text-on-surface-variant/70">
            Yeni Şifre
          </label>
          <input
            id="yeni-sifre"
            v-model="sifreForm.new_password"
            type="password"
            autocomplete="new-password"
            required
            class="w-full border-b border-outline-variant bg-transparent py-2 text-body-md text-on-surface transition-colors focus:border-accent-gold focus:outline-none"
          >
        </div>
      </div>

      <div class="mt-9 flex flex-wrap items-center gap-4">
        <button type="submit" :disabled="sifreKaydediliyor" class="btn-primary text-label-caps disabled:opacity-60">
          {{ sifreKaydediliyor ? 'Kaydediliyor...' : 'Şifreyi Güncelle' }}
        </button>
        <p v-if="sifreMesaj" class="text-sm text-on-surface-variant" role="status">
          {{ sifreMesaj }}
        </p>
        <p v-if="sifreHata" class="text-sm text-red-700" role="alert">
          {{ sifreHata }}
        </p>
      </div>
    </form>

    <!-- Hesap işlemleri -->
    <div class="mt-10 border-t border-outline-variant/40 pt-8">
      <h2 class="text-label-caps text-on-surface-variant/70">
        Hesap İşlemleri
      </h2>
      <div class="mt-5 flex flex-wrap items-center gap-4">
        <button type="button" :disabled="cikisYapiliyor" class="btn-secondary text-label-caps disabled:opacity-60" @click="cikisYap">
          <Icon name="material-symbols:logout" size="16" />
          {{ cikisYapiliyor ? 'Çıkış yapılıyor...' : 'Çıkış Yap' }}
        </button>
      </div>
    </div>
  </div>
</template>
