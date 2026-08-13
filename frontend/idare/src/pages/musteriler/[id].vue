<script setup lang="ts">
import { useCustomers } from '@/composables/useCustomers'
import type { CustomerDetail } from '@/model/customer'
import { STATUS_COLORS, STATUS_LABELS } from '@/model/order'
import { formatTutar as tutar } from '@/utils/Currency'
import { ErrorPopup } from '@/utils/Popup'

const route = useRoute('musteriler-id')
const api = useCustomers()

const loading = ref(false)
const customer = ref<CustomerDetail | null>(null)

const headers = [
  { title: 'Sipariş No', key: 'order_no', width: 130 },
  { title: 'Tarih', key: 'created_at', width: 160 },
  { title: 'Tutar', key: 'total', width: 120 },
  { title: 'Durum', key: 'status', width: 130 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.get(Number(route.params.id))

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  customer.value = data
}

onMounted(load)

const router = useRouter()

function handleRowClick(_: unknown, payload: { item: { id: number } }) {
  router.push(`/siparisler/${payload.item.id}`)
}

const tarih = (d: string) =>
  new Date(d).toLocaleString('tr-TR', { dateStyle: 'short', timeStyle: 'short' })

const tarihKisa = (d: string) => new Date(d).toLocaleDateString('tr-TR')
</script>

<template>
  <div v-if="customer">
    <div class="d-flex align-center gap-2 mb-6">
      <VBtn
        icon="tabler-arrow-left"
        variant="text"
        to="/musteriler"
      />
      <h4 class="text-h4">
        {{ customer.name }}
      </h4>
    </div>

    <VRow>
      <VCol
        cols="12"
        md="4"
      >
        <VCard>
          <VCardItem>
            <VCardTitle>Profil</VCardTitle>
          </VCardItem>
          <VCardText>
            <p><strong>Ad Soyad:</strong> {{ customer.name }}</p>
            <p><strong>E-posta:</strong> {{ customer.email }}</p>
            <p><strong>Telefon:</strong> {{ customer.phone }}</p>
            <p><strong>Kayıt Tarihi:</strong> {{ tarihKisa(customer.created_at) }}</p>
          </VCardText>
        </VCard>
      </VCol>

      <VCol
        cols="12"
        md="8"
      >
        <VCard>
          <VCardItem>
            <VCardTitle>Sipariş Geçmişi</VCardTitle>
          </VCardItem>

          <VDataTable
            :headers="headers"
            :items="customer.orders"
            :loading="loading"
            :items-per-page="-1"
            no-data-text="Sipariş yok"
            loading-text="Yükleniyor..."
            hover
            @click:row="handleRowClick"
          >
            <template #item.order_no="{ item }">
              <code class="text-caption">{{ item.order_no }}</code>
            </template>

            <template #item.created_at="{ item }">
              {{ tarih(item.created_at) }}
            </template>

            <template #item.total="{ item }">
              {{ tutar(item.total) }}
            </template>

            <template #item.status="{ item }">
              <VChip
                :color="STATUS_COLORS[item.status]"
                size="small"
              >
                {{ STATUS_LABELS[item.status] }}
              </VChip>
            </template>
          </VDataTable>
        </VCard>
      </VCol>
    </VRow>
  </div>
</template>
