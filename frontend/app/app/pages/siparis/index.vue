<script setup lang="ts">
import { formatPrice } from '~/utils/price'
import { apiErrorMessage } from '~/composables/useOrders'

const { items, itemsTotal, clear } = useCart()
const { data: cfg } = await useDeliveryConfig()
const router = useRouter()

// Sepet boşken bu sayfanın anlamı yok (spec §5 kenar durumlar)
onMounted(() => {
  if (!items.value.length)
    router.replace('/urunler')
})

const form = reactive({
  buyerName: '',
  buyerPhone: '',
  buyerEmail: '',
  aliciBenim: false,
  recipientName: '',
  recipientPhone: '',
  address: '',
  date: '',
  slot: '',
  cardMessage: '',
})

// "Alıcı benim" — form uzunluğu dönüşümü düşürür (MVP §3.3), tek tık kısayol
watch(() => form.aliciBenim, (v) => {
  if (v) {
    form.recipientName = form.buyerName
    form.recipientPhone = form.buyerPhone
  }
})

const gonderiliyor = ref(false)
const hata = ref('')

const bugun = new Date().toISOString().slice(0, 10)
const sonTarih = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() + (cfg.value?.max_days ?? 30))

  return d.toISOString().slice(0, 10)
})

const toplam = computed(() => {
  const ara = Number.parseFloat(itemsTotal.value)
  const ucret = Number.parseFloat(cfg.value?.fee ?? '0')

  return (ara + ucret).toFixed(2)
})

async function gonder() {
  hata.value = ''
  gonderiliyor.value = true

  try {
    const sonuc = await createOrder({
      items: items.value.map(i => ({ product_id: i.product_id, quantity: i.quantity })),
      buyer: { name: form.buyerName, phone: form.buyerPhone, email: form.buyerEmail || undefined },
      recipient: { name: form.recipientName, phone: form.recipientPhone },
      delivery: { address: form.address, date: form.date, slot: form.slot },
      card_message: form.cardMessage || undefined,
    })

    // Başarılı → sepeti temizle, yoksa müşteri aynı siparişi tekrar verebilir
    clear()
    await router.push(`/siparis/tamam?no=${sonuc.order_no}`)
  }
  catch (e) {
    // Başarısız → sepet KORUNUR, müşterinin emeği silinmesin (spec §5)
    hata.value = apiErrorMessage(e)
  }
  finally {
    gonderiliyor.value = false
  }
}

useSeoMeta({
  title: 'Siparişi Tamamla | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <h1 class="font-serif text-3xl text-primary md:text-4xl">
      Siparişi Tamamla
    </h1>

    <form class="mt-10 grid gap-12 lg:grid-cols-[1fr_380px]" @submit.prevent="gonder">
      <div class="space-y-10">
        <!-- Sipariş veren -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Sipariş Veren
          </legend>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Ad Soyad *</span>
              <input v-model="form.buyerName" required class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Telefon *</span>
              <input v-model="form.buyerPhone" required type="tel" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">E-posta</span>
              <input v-model="form.buyerEmail" type="email" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
          </div>
        </fieldset>

        <!-- Alıcı -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Alıcı
          </legend>

          <label class="mt-4 flex items-center gap-2">
            <input v-model="form.aliciBenim" type="checkbox" class="size-4">
            <span class="text-body-md text-on-surface-variant">Alıcı benim</span>
          </label>

          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Alıcı Adı *</span>
              <input v-model="form.recipientName" required class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Alıcı Telefonu *</span>
              <input v-model="form.recipientPhone" required type="tel" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">Teslimat Adresi *</span>
              <textarea v-model="form.address" required rows="3" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none" />
            </label>
          </div>
        </fieldset>

        <!-- Teslimat -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Teslimat
          </legend>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Tarih *</span>
              <input v-model="form.date" required type="date" :min="bugun" :max="sonTarih" class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Saat *</span>
              <select v-model="form.slot" required class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none">
                <option value="" disabled>
                  Seçiniz
                </option>
                <option v-for="s in cfg?.slots ?? []" :key="s" :value="s">
                  {{ s }}
                </option>
              </select>
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">Kart Mesajı</span>
              <textarea
                v-model="form.cardMessage"
                rows="3"
                placeholder="Doğum günün kutlu olsun!"
                class="mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5 text-body-md text-on-surface focus:border-secondary focus:outline-none"
              />
            </label>
          </div>
        </fieldset>
      </div>

      <!-- Özet -->
      <aside class="h-fit rounded-lg border border-outline-variant/40 bg-surface-container-low p-6">
        <h2 class="font-serif text-xl text-primary">
          Sipariş Özeti
        </h2>

        <ul class="mt-5 space-y-3">
          <li v-for="item in items" :key="item.product_id" class="flex justify-between gap-3 text-body-md">
            <span class="text-on-surface-variant">{{ item.name }} × {{ item.quantity }}</span>
            <span class="shrink-0 text-primary">{{ formatPrice(item.price) }}</span>
          </li>
        </ul>

        <div class="mt-5 space-y-2 border-t border-outline-variant/30 pt-4 text-body-md">
          <div class="flex justify-between">
            <span class="text-on-surface-variant">Ara Toplam</span>
            <span>{{ formatPrice(itemsTotal) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-on-surface-variant">Teslimat</span>
            <span>{{ formatPrice(cfg?.fee ?? '0') }}</span>
          </div>
        </div>

        <div class="mt-4 flex justify-between border-t border-outline-variant/30 pt-4">
          <span class="font-serif text-lg text-primary">Toplam</span>
          <span class="font-serif text-lg text-primary">{{ formatPrice(toplam) }}</span>
        </div>

        <p v-if="hata" class="mt-4 text-body-md text-red-700" role="alert">
          {{ hata }}
        </p>

        <button
          type="submit"
          :disabled="gonderiliyor"
          class="btn-primary text-label-caps mt-6 w-full disabled:opacity-60"
        >
          {{ gonderiliyor ? 'Gönderiliyor...' : 'Siparişi Gönder' }}
        </button>

        <p class="mt-3 text-xs text-on-surface-variant">
          Siparişiniz alındıktan sonra sizinle iletişime geçilecektir.
        </p>
      </aside>
    </form>
  </div>
</template>
