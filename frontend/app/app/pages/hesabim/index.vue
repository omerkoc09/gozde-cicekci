<script setup lang="ts">
import type { Customer, CustomerOrder, CustomerOrderStatus } from '~/types/api'
import { apiErrorMessage } from '~/composables/useOrders'
import { formatPrice } from '~/utils/price'

definePageMeta({ layout: 'account' })

useSeoMeta({ title: 'Hesap Panom | Gözde Tasarım Çiçekçilik' })

const { me, myOrders } = useCustomer()
const router = useRouter()

const musteri = ref<Customer | null>(null)
const siparisler = ref<CustomerOrder[]>([])
const yukleniyor = ref(true)
const hata = ref('')

onMounted(async () => {
  const sonuc = await me()
  if (!sonuc) {
    await router.replace('/giris')
    return
  }
  musteri.value = sonuc

  try {
    siparisler.value = await myOrders()
  }
  catch (e) {
    hata.value = apiErrorMessage(e)
  }
  finally {
    yukleniyor.value = false
  }
})

const DURUM_ETIKET: Record<CustomerOrderStatus, string> = {
  awaiting_payment: 'Ödeme Bekleniyor',
  paid: 'Ödendi',
  delivered: 'Teslim Edildi',
  refunded: 'İade Edildi',
}

const DURUM_RENK: Record<CustomerOrderStatus, string> = {
  awaiting_payment: 'bg-surface-container-low text-on-surface-variant',
  paid: 'bg-secondary/10 text-secondary',
  delivered: 'bg-primary/10 text-primary',
  refunded: 'bg-error-container/40 text-error',
}

function formatTarih(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()))
    return iso

  return d.toLocaleDateString('tr-TR', { day: 'numeric', month: 'long', year: 'numeric' })
}
</script>

<template>
  <div>
    <AccountHero
      v-if="musteri"
      :title="`Merhaba ${musteri.name}`"
      description="Hesap panonuzdan sipariş geçmişinizi görüntüleyebilir ve hesap detaylarınızı düzenleyebilirsiniz."
    />

    <section class="mt-10">
      <h2 class="font-serif text-xl text-primary">
        Sipariş Geçmişi
      </h2>

      <p v-if="yukleniyor" class="mt-4 text-body-md text-on-surface-variant">
        Yükleniyor...
      </p>

      <p v-else-if="hata" class="mt-4 text-body-md text-red-700" role="alert">
        {{ hata }}
      </p>

      <p v-else-if="!siparisler.length" class="mt-4 text-body-md text-on-surface-variant">
        Henüz siparişiniz yok.
      </p>

      <ul v-else class="mt-6 space-y-4">
        <li
          v-for="s in siparisler"
          :key="s.order_no"
          class="rounded-lg border border-outline-variant/40 bg-surface-container-lowest p-5"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="font-serif text-lg text-primary">
                {{ s.order_no }}
              </p>
              <p class="mt-1 text-body-md text-on-surface-variant">
                {{ formatTarih(s.delivery_date) }}
              </p>
            </div>

            <div class="flex items-center gap-3">
              <span
                class="text-label-caps rounded-full px-3 py-1"
                :class="DURUM_RENK[s.status]"
              >
                {{ DURUM_ETIKET[s.status] }}
              </span>
              <span class="font-serif text-lg text-primary">
                {{ formatPrice(s.total) }}
              </span>
            </div>
          </div>

          <ul v-if="s.items.length" class="mt-4 space-y-1 border-t border-outline-variant/30 pt-3">
            <li v-for="(item, i) in s.items" :key="i" class="text-body-md text-on-surface-variant">
              {{ item.product_name }} × {{ item.quantity }}
            </li>
          </ul>
        </li>
      </ul>
    </section>
  </div>
</template>
