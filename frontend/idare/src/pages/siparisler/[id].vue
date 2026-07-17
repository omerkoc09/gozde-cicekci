<script setup lang="ts">
import { useOrders } from '@/composables/useOrders'
import type { Order, OrderStatus } from '@/model/order'
import { STATUS_COLORS, STATUS_LABELS } from '@/model/order'
import { ErrorPopup, SuccessToast } from '@/utils/Popup'

const route = useRoute('siparisler-id')
const api = useOrders()

const loading = ref(false)
const saving = ref(false)
const order = ref<Order | null>(null)
const note = ref('')

const load = async () => {
  loading.value = true

  const [err, data] = await api.get(Number(route.params.id))

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  order.value = data
  note.value = data.note
}

onMounted(load)

const setStatus = async (status: OrderStatus) => {
  saving.value = true

  const [err] = await api.update(Number(route.params.id), { status })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Durum güncellendi')
  await load()
}

const saveNote = async () => {
  saving.value = true

  const [err] = await api.update(Number(route.params.id), { note: note.value })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Not kaydedildi')
}

const tutar = (v: string) =>
  `${Number.parseFloat(v).toLocaleString('tr-TR', { minimumFractionDigits: 2 })} ₺`
</script>

<template>
  <div v-if="order">
    <div class="d-flex align-center gap-2 mb-6">
      <VBtn
        icon="tabler-arrow-left"
        variant="text"
        to="/siparisler"
      />
      <h4 class="text-h4">
        {{ order.order_no }}
      </h4>
      <VChip :color="STATUS_COLORS[order.status]">
        {{ STATUS_LABELS[order.status] }}
      </VChip>
    </div>

    <VRow>
      <VCol
        cols="12"
        md="7"
      >
        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Ürünler</VCardTitle>
          </VCardItem>
          <VCardText>
            <div
              v-for="item in order.items"
              :key="item.product_name"
              class="d-flex justify-space-between py-2 border-b"
            >
              <span>{{ item.product_name }} × {{ item.quantity }}</span>
              <span>{{ tutar(item.price_at_order) }}</span>
            </div>

            <div class="d-flex justify-space-between pt-4">
              <span class="text-medium-emphasis">Ara Toplam</span>
              <span>{{ tutar(order.items_total) }}</span>
            </div>
            <div class="d-flex justify-space-between">
              <span class="text-medium-emphasis">Teslimat</span>
              <span>{{ tutar(order.delivery_fee) }}</span>
            </div>
            <div class="d-flex justify-space-between text-h6 pt-2">
              <span>Toplam</span>
              <span>{{ tutar(order.total) }}</span>
            </div>
          </VCardText>
        </VCard>

        <VCard>
          <VCardItem>
            <VCardTitle>Teslimat</VCardTitle>
          </VCardItem>
          <VCardText>
            <p><strong>Alıcı:</strong> {{ order.recipient_name }} — {{ order.recipient_phone }}</p>
            <p><strong>Adres:</strong> {{ order.delivery_address }} ({{ order.delivery_district }})</p>
            <p><strong>Tarih:</strong> {{ order.delivery_date }} · {{ order.delivery_slot }}</p>
            <VAlert
              v-if="order.card_message"
              type="info"
              variant="tonal"
              class="mt-4"
            >
              <strong>Kart mesajı:</strong> {{ order.card_message }}
            </VAlert>
          </VCardText>
        </VCard>
      </VCol>

      <VCol
        cols="12"
        md="5"
      >
        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Sipariş Veren</VCardTitle>
          </VCardItem>
          <VCardText>
            <p>{{ order.buyer_name }}</p>
            <p>{{ order.buyer_phone }}</p>
            <p v-if="order.buyer_email">
              {{ order.buyer_email }}
            </p>
          </VCardText>
        </VCard>

        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Durum</VCardTitle>
          </VCardItem>
          <VCardText class="d-flex flex-column gap-2">
            <VBtn
              v-if="order.status === 'pending'"
              :loading="saving"
              @click="setStatus('confirmed')"
            >
              Onayla
            </VBtn>
            <VBtn
              v-if="order.status === 'confirmed'"
              :loading="saving"
              color="success"
              @click="setStatus('delivered')"
            >
              Teslim Edildi
            </VBtn>
            <VBtn
              v-if="order.status !== 'cancelled' && order.status !== 'delivered'"
              :loading="saving"
              color="error"
              variant="outlined"
              @click="setStatus('cancelled')"
            >
              İptal Et
            </VBtn>
          </VCardText>
        </VCard>

        <VCard>
          <VCardItem>
            <VCardTitle>Not</VCardTitle>
          </VCardItem>
          <VCardText>
            <VTextarea
              v-model="note"
              rows="3"
              placeholder="Kendi notunuz..."
            />
            <VBtn
              :loading="saving"
              class="mt-2"
              @click="saveNote"
            >
              Kaydet
            </VBtn>
          </VCardText>
        </VCard>
      </VCol>
    </VRow>
  </div>
</template>
