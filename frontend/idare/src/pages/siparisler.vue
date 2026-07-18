<script setup lang="ts">
import { useOrders } from '@/composables/useOrders'
import type { Order } from '@/model/order'
import { STATUS_COLORS, STATUS_LABELS } from '@/model/order'
import { formatTutar as tutar } from '@/utils/Currency'
import { ErrorPopup } from '@/utils/Popup'

const api = useOrders()
const router = useRouter()

const loading = ref(false)
const orders = ref<Order[]>([])
const statusFilter = ref<string>('')

const headers = [
  { title: 'Sipariş No', key: 'order_no', width: 130 },
  { title: 'Alıcı', key: 'recipient_name' },
  { title: 'Teslimat', key: 'delivery_date', width: 180 },
  { title: 'Tutar', key: 'total', width: 120 },
  { title: 'Durum', key: 'status', width: 130 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list(statusFilter.value)

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  orders.value = data ?? []
}

onMounted(load)
watch(statusFilter, load)

const tarih = (d: string, slot: string) => {
  const [y, m, g] = d.split('-')

  return `${g}.${m}.${y} · ${slot}`
}

function handleRowClick(_: unknown, payload: { item: Order }) {
  router.push(`/siparisler/${payload.item.id}`)
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Siparişler
      </h4>

      <VBtnToggle
        v-model="statusFilter"
        density="compact"
        divided
      >
        <VBtn value="">
          Hepsi
        </VBtn>
        <VBtn value="pending">
          Yeni
        </VBtn>
        <VBtn value="confirmed">
          Onaylandı
        </VBtn>
        <VBtn value="delivered">
          Teslim
        </VBtn>
      </VBtnToggle>
    </div>

    <VCard>
      <VDataTable
        :headers="headers"
        :items="orders"
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

        <template #item.recipient_name="{ item }">
          <div class="font-weight-medium">
            {{ item.recipient_name }}
          </div>
          <div class="text-caption text-medium-emphasis">
            {{ item.items.map(i => `${i.product_name} ×${i.quantity}`).join(', ') }}
          </div>
        </template>

        <template #item.delivery_date="{ item }">
          {{ tarih(item.delivery_date, item.delivery_slot) }}
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
  </div>
</template>
