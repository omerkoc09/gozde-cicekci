<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useCustomers } from '@/composables/useCustomers'
import type { Customer } from '@/model/customer'
import { ErrorPopup } from '@/utils/Popup'

const api = useCustomers()
const router = useRouter()

const loading = ref(false)
const customers = ref<Customer[]>([])
const total = ref(0)
const search = ref('')

const page = ref(1)
const itemsPerPage = ref(25)

const headers = [
  { title: 'Ad Soyad', key: 'name' },
  { title: 'E-posta', key: 'email' },
  { title: 'Telefon', key: 'phone' },
  { title: 'Kayıt Tarihi', key: 'created_at', width: 180 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list({
    q: search.value,
    page: page.value,
    limit: itemsPerPage.value,
  })

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  customers.value = data?.items ?? []
  total.value = data?.total ?? 0
}

onMounted(load)

const debouncedSearch = useDebounceFn(() => {
  page.value = 1
  load()
}, 400)

watch(search, debouncedSearch)

function handleOptionsUpdate(options: { page: number; itemsPerPage: number }) {
  page.value = options.page
  itemsPerPage.value = options.itemsPerPage
  load()
}

function handleRowClick(_: unknown, payload: { item: Customer }) {
  router.push(`/musteriler/${payload.item.id}`)
}

const tarih = (d: string) => new Date(d).toLocaleDateString('tr-TR')
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Müşteriler
      </h4>
    </div>

    <VCard>
      <VCardText>
        <VTextField
          v-model="search"
          prepend-inner-icon="tabler-search"
          label="Ad veya e-posta ile ara..."
          density="compact"
          clearable
          hide-details
          style="max-width: 360px;"
        />
      </VCardText>

      <VDataTableServer
        :headers="headers"
        :items="customers"
        :items-length="total"
        :loading="loading"
        :items-per-page="itemsPerPage"
        :page="page"
        no-data-text="Müşteri yok"
        loading-text="Yükleniyor..."
        hover
        @update:options="handleOptionsUpdate"
        @click:row="handleRowClick"
      >
        <template #item.created_at="{ item }">
          {{ tarih(item.created_at) }}
        </template>
      </VDataTableServer>
    </VCard>
  </div>
</template>
