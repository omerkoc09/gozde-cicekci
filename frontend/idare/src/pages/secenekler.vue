<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import { useOptions } from '@/composables/useOptions'
import type { GroupProduct, OptionGroup, OptionKind, OptionValue } from '@/model/option'
import { KIND_LABELS } from '@/model/option'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'
import { requiredValidator } from '@validators'

const api = useOptions()

const loading = ref(false)
const groups = ref<OptionGroup[]>([])

// Sağ sütunda gösterilen grup — satıra tıklayınca değişir.
const seciliGrupId = ref<number | null>(null)

const seciliGrup = computed(() =>
  groups.value.find(g => g.id === seciliGrupId.value) ?? null)

const sirali = computed(() =>
  [...groups.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id))

const siraliDegerler = computed(() =>
  seciliGrup.value
    ? [...seciliGrup.value.values].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    : [])

// Kolon sıralaması kapalı: ok butonları satır index'ine dayanıyor, tablo
// yeniden sıralanırsa yanlış çift takas edilir. TÜM kolonlarda sortable:
// false olmalı — tek bir sıralanabilir başlık bile move()'u bozar.
const groupHeaders = [
  { title: 'Ad', key: 'name', sortable: false },
  { title: 'Tip', key: 'kind', sortable: false, width: 100 },
  { title: 'Seçenek', key: 'value_count', sortable: false, width: 100 },
  { title: 'Aktif', key: 'is_active', sortable: false, width: 110 },
  { title: 'Sıra', key: 'sira', sortable: false, width: 110 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 110 },
]

// Aynı sebep: moveValue() de satır index'ine dayanıyor.
const valueHeaders = [
  { title: 'Renk', key: 'swatch', sortable: false, width: 70 },
  { title: 'Ad', key: 'name', sortable: false },
  { title: 'Aktif', key: 'is_active', sortable: false, width: 110 },
  { title: 'Sıra', key: 'sira', sortable: false, width: 110 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 70 },
]

const arama = ref('')

const siralaniyor = ref(false)
const degerSiralaniyor = ref(false)

/**
 * Grubu bir sıra kaydırır. Backend TÜM grupların id'sini istiyor — yeni
 * sıralamanın tamamı gönderiliyor.
 *
 * Arama doluyken kilitli: filtrelenmiş listede komşu görünen iki satır
 * gerçek listede komşu olmayabilir — takas şaşırtıcı sonuç doğurur.
 */
const move = async (index: number, direction: -1 | 1) => {
  const next = index + direction
  if (next < 0 || next >= sirali.value.length)
    return

  const ids = sirali.value.map(g => g.id)

  ;[ids[index], ids[next]] = [ids[next], ids[index]]

  siralaniyor.value = true

  const [err, guncel] = await api.reorderGroups(ids)

  siralaniyor.value = false

  if (err)
    return ErrorPopup(err.message)

  // Uç güncel listenin tamamını dönüyor — yeniden yüklemeye gerek yok.
  groups.value = guncel ?? []
}

/** Seçili grubun değerlerini bir sıra kaydırır. */
const moveValue = async (index: number, direction: -1 | 1) => {
  if (!seciliGrup.value)
    return

  const liste = siraliDegerler.value
  const next = index + direction
  if (next < 0 || next >= liste.length)
    return

  const ids = liste.map(v => v.id)

  ;[ids[index], ids[next]] = [ids[next], ids[index]]

  degerSiralaniyor.value = true

  const [err, guncel] = await api.reorderValues(seciliGrup.value.id, ids)

  degerSiralaniyor.value = false

  if (err)
    return ErrorPopup(err.message)

  groups.value = guncel ?? []
}

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  groups.value = data ?? []
}

onMounted(load)

const seciliGrubuGoster = (g: OptionGroup) => {
  seciliGrupId.value = g.id
}

// --- Grup oluştur/düzenle diyaloğu ---

const dialog = ref(false)
const saving = ref(false)
const formRef = ref<VForm>()

// Düzenlenen grup; null ise yeni kayıt.
const editing = ref<OptionGroup | null>(null)

// sort_order YOK: sıra formdan değil, listedeki ok butonlarından değişir.
const form = ref({
  name: '',
  kind: 'color' as OptionKind,
  is_active: true,
})

const openCreate = () => {
  editing.value = null
  form.value = { name: '', kind: 'color', is_active: true }
  dialog.value = true
}

const openEdit = (g: OptionGroup) => {
  editing.value = g
  form.value = {
    name: g.name,
    kind: g.kind,
    is_active: g.is_active,
  }
  dialog.value = true
}

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  saving.value = true

  // kind yalnızca oluştururken gönderilir — sonradan değiştirilemez.
  const [err] = editing.value
    ? await api.updateGroup(editing.value.id, {
      name: form.value.name,
      is_active: form.value.is_active,
    })
    : await api.createGroup({ name: form.value.name, kind: form.value.kind })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  dialog.value = false
  SuccessToast(editing.value ? 'Grup güncellendi' : 'Grup oluşturuldu')
  await load()
}

/**
 * Switch anında kaydeder. İyimser değil: yanıt gelene kadar satırdaki
 * değer eski kalır, hata olursa hiç değişmez.
 */
const toggleGroup = async (g: OptionGroup, value: boolean) => {
  const [err] = await api.updateGroup(g.id, { is_active: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

const removeGroup = async (g: OptionGroup) => {
  const [err, data] = await api.groupProductCount(g.id)
  if (err)
    return ErrorPopup(err.message)

  const n = data.product_count

  const ok = await ConfirmPopup(
    n > 0
      ? `"${g.name}" grubu ${n} üründe kullanılıyor. Silerseniz bu ürünlerde artık sorulmayacak (geçmiş siparişler etkilenmez). Devam edilsin mi?`
      : `"${g.name}" grubu silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [delErr] = await api.removeGroup(g.id)
  if (delErr)
    return ErrorPopup(delErr.message)

  if (seciliGrupId.value === g.id)
    seciliGrupId.value = null

  SuccessToast('Grup silindi')
  await load()
}

// --- Sağ sütun: yeni seçenek formu ---

const valueForm = ref({ name: '', swatch_hex: '#000000' })
const valueSaving = ref(false)
const valueFormRef = ref<VForm>()

watch(seciliGrupId, () => {
  valueForm.value = { name: '', swatch_hex: '#000000' }
})

// --- Sağ sütun: bu grubu kullanan ürünler ---

const kullananUrunler = ref<GroupProduct[]>([])
const urunlerYukleniyor = ref(false)

/** Sağ paneldeki aktif sekme. Grup değişince "Seçenekler"e döner. */
const sagSekme = ref<'degerler' | 'urunler'>('degerler')

watch(seciliGrupId, () => {
  sagSekme.value = 'degerler'
})

/**
 * Grup değişince listeyi tazeler. Ayrı uçtan geliyor çünkü grup listesi
 * (list()) ürün bilgisini taşımıyor — her grup için ürünleri de çekmek
 * sayfa açılışında N+1 olurdu.
 */
watch(seciliGrupId, async id => {
  kullananUrunler.value = []
  if (id === null)
    return

  urunlerYukleniyor.value = true

  const [err, data] = await api.groupProducts(id)

  urunlerYukleniyor.value = false

  if (err)
    return ErrorPopup(err.message)

  // Yanıt geldiğinde kullanıcı başka gruba geçmiş olabilir — geç gelen
  // yanıt yanlış grubun listesini yazmasın.
  if (seciliGrupId.value === id)
    kullananUrunler.value = data ?? []
}, { immediate: true })

const addValue = async () => {
  if (!seciliGrup.value)
    return

  const { valid } = await valueFormRef.value!.validate()
  if (!valid)
    return

  valueSaving.value = true

  const [err] = await api.createValue(seciliGrup.value.id, {
    name: valueForm.value.name,
    swatch_hex: seciliGrup.value.kind === 'color' ? valueForm.value.swatch_hex : '',
  })

  valueSaving.value = false

  if (err)
    return ErrorPopup(err.message)

  valueForm.value = { name: '', swatch_hex: '#000000' }
  SuccessToast('Seçenek eklendi')
  await load()
}

const toggleValue = async (v: OptionValue, value: boolean) => {
  const [err] = await api.updateValue(v.id, { is_active: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

const removeValue = async (v: OptionValue) => {
  const ok = await ConfirmPopup(
    `"${v.name}" seçeneği silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [err] = await api.removeValue(v.id)
  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Seçenek silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Seçenek Yönetimi
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="openCreate"
      >
        Yeni Grup
      </VBtn>
    </div>

    <VAlert
      type="info"
      variant="tonal"
      class="mb-6"
    >
      Buradaki gruplar ürün formunda "Özelleştirme" bölümünde çıkar; her ürün
      için ayrı ayrı açılır. Pasif grup veya seçenek müşteriye gösterilmez,
      geçmiş siparişler etkilenmez. Sırayı değiştirmek için yukarı/aşağı
      oklarını kullanın.
    </VAlert>

    <VRow>
      <VCol
        cols="12"
        md="7"
      >
        <VCard>
          <VCardItem>
            <VCardTitle>Gruplar</VCardTitle>
          </VCardItem>

          <VCardText>
            <VTextField
              v-model="arama"
              prepend-inner-icon="tabler-search"
              placeholder="Grup ara..."
              class="mb-4"
              hide-details
              clearable
            />
          </VCardText>

          <VDataTable
            :headers="groupHeaders"
            :items="sirali"
            :search="arama"
            :loading="loading"
            items-per-page="-1"
            no-data-text="Bu aramaya uyan grup yok"
            loading-text="Yükleniyor..."
          >
            <template #item.name="{ item }">
              <span
                class="cursor-pointer"
                :class="{ 'text-disabled': !item.is_active, 'font-weight-bold': item.id === seciliGrupId }"
                @click="seciliGrubuGoster(item)"
              >{{ item.name }}</span>
            </template>

            <template #item.kind="{ item }">
              <VChip
                size="small"
                :color="item.kind === 'color' ? 'primary' : 'secondary'"
              >
                {{ KIND_LABELS[item.kind] }}
              </VChip>
            </template>

            <template #item.value_count="{ item }">
              {{ item.values.length }}
            </template>

            <template #item.is_active="{ item }">
              <VSwitch
                :model-value="item.is_active"
                density="compact"
                hide-details
                @update:model-value="toggleGroup(item, $event as boolean)"
              />
            </template>

            <!--
              Sıra artık sayı olarak girilmiyor: esnaf ok butonlarıyla
              taşıyor, sunucu sort_order'ı kendisi hesaplıyor. index sıralı
              listedeki konum — VDataTable items'ı sirali ile aynı sırada
              aldığı için move() de aynı diziyi kullanabiliyor.
            -->
            <template #item.sira="{ index }">
              <VBtn
                icon="tabler-chevron-up"
                variant="text"
                size="x-small"
                :disabled="siralaniyor || !!arama || index === 0"
                :title="arama ? 'Sıralamak için aramayı temizleyin' : 'Yukarı taşı'"
                @click="move(index, -1)"
              />
              <VBtn
                icon="tabler-chevron-down"
                variant="text"
                size="x-small"
                :disabled="siralaniyor || !!arama || index === sirali.length - 1"
                :title="arama ? 'Sıralamak için aramayı temizleyin' : 'Aşağı taşı'"
                @click="move(index, 1)"
              />
            </template>

            <template #item.actions="{ item }">
              <VBtn
                icon="tabler-pencil"
                variant="text"
                size="small"
                @click="openEdit(item)"
              />
              <VBtn
                icon="tabler-trash"
                variant="text"
                size="small"
                color="error"
                @click="removeGroup(item)"
              />
            </template>
          </VDataTable>
        </VCard>
      </VCol>

      <VCol
        cols="12"
        md="5"
      >
        <VCard>
          <VCardItem>
            <VCardTitle>
              {{ seciliGrup ? seciliGrup.name : 'Seçenekler' }}
            </VCardTitle>
          </VCardItem>

          <!--
            İki sekme: grubun değerleri ve grubu kullanan ürünler. Sekme
            olarak duruyorlar çünkü ikisi de aynı grubun iki ayrı yüzü —
            alt alta konsaydı sağ sütun çok uzar, ürün listesi ekranın
            dışında kalırdı.
          -->
          <VTabs
            v-if="seciliGrup"
            v-model="sagSekme"
            density="compact"
            class="px-4"
          >
            <VTab value="degerler">
              Seçenekler
              <VChip
                size="x-small"
                class="ms-2"
              >
                {{ seciliGrup.values.length }}
              </VChip>
            </VTab>
            <VTab value="urunler">
              Kullanan Ürünler
              <VChip
                v-if="!urunlerYukleniyor"
                size="x-small"
                class="ms-2"
              >
                {{ kullananUrunler.length }}
              </VChip>
            </VTab>
          </VTabs>

          <VDivider v-if="seciliGrup" />

          <VCardText v-if="!seciliGrup">
            <VAlert
              type="info"
              variant="tonal"
            >
              Soldan bir grup seçin
            </VAlert>
          </VCardText>

          <template v-else-if="sagSekme === 'degerler'">
            <VDataTable
              :headers="valueHeaders"
              :items="siraliDegerler"
              items-per-page="-1"
              no-data-text="Henüz seçenek yok"
              density="compact"
            >
              <template #item.swatch="{ item }">
                <div
                  v-if="seciliGrup!.kind === 'color'"
                  class="rounded-circle border"
                  style="width: 22px; height: 22px;"
                  :style="{ background: item.swatch_hex }"
                />
              </template>

              <template #item.name="{ item }">
                <span :class="{ 'text-disabled': !item.is_active }">{{ item.name }}</span>
              </template>

              <template #item.is_active="{ item }">
                <VSwitch
                  :model-value="item.is_active"
                  density="compact"
                  hide-details
                  @update:model-value="toggleValue(item, $event as boolean)"
                />
              </template>

              <template #item.sira="{ index }">
                <VBtn
                  icon="tabler-chevron-up"
                  variant="text"
                  size="x-small"
                  :disabled="degerSiralaniyor || index === 0"
                  title="Yukarı taşı"
                  @click="moveValue(index, -1)"
                />
                <VBtn
                  icon="tabler-chevron-down"
                  variant="text"
                  size="x-small"
                  :disabled="degerSiralaniyor || index === siraliDegerler.length - 1"
                  title="Aşağı taşı"
                  @click="moveValue(index, 1)"
                />
              </template>

              <template #item.actions="{ item }">
                <VBtn
                  icon="tabler-trash"
                  variant="text"
                  size="small"
                  color="error"
                  @click="removeValue(item)"
                />
              </template>
            </VDataTable>

            <VDivider />

            <VCardText>
              <VForm
                ref="valueFormRef"
                @submit.prevent="addValue"
              >
                <VRow align="center">
                  <VCol
                    cols="12"
                    :sm="seciliGrup.kind === 'color' ? 5 : 8"
                  >
                    <VTextField
                      v-model="valueForm.name"
                      label="Seçenek Adı"
                      density="compact"
                      :rules="[requiredValidator]"
                      hide-details="auto"
                    />
                  </VCol>

                  <template v-if="seciliGrup.kind === 'color'">
                    <VCol
                      cols="4"
                      sm="2"
                    >
                      <VTextField
                        v-model="valueForm.swatch_hex"
                        type="color"
                        label="Renk"
                        density="compact"
                        hide-details
                      />
                    </VCol>
                    <VCol
                      cols="8"
                      sm="3"
                    >
                      <VTextField
                        v-model="valueForm.swatch_hex"
                        label="Hex"
                        density="compact"
                        placeholder="#RRGGBB"
                        hide-details="auto"
                      />
                    </VCol>
                  </template>

                  <VCol
                    cols="12"
                    sm="2"
                  >
                    <VBtn
                      block
                      :loading="valueSaving"
                      @click="addValue"
                    >
                      Ekle
                    </VBtn>
                  </VCol>
                </VRow>
              </VForm>
            </VCardText>
          </template>

          <!--
            Bu grup hangi ürünlerde soruluyor. Esnaf "Ambalaj Rengi'ni
            nerelerde kullanıyorum?" sorusunu ürünleri tek tek açmadan
            cevaplayabilsin diye. Pasif ürünler de listeleniyor (soluk) —
            silmeden önce tam tablo görünmeli.
          -->
          <template v-else>
            <VProgressLinear
              v-if="urunlerYukleniyor"
              indeterminate
            />

            <VCardText v-else-if="!kullananUrunler.length">
              <span class="text-medium-emphasis text-body-2">
                Bu grup henüz hiçbir üründe açık değil. Ürün formundaki
                "Özelleştirme" bölümünden aktifleştirebilirsiniz.
              </span>
            </VCardText>

            <VList
              v-else
              density="compact"
              class="py-0"
            >
              <VListItem
                v-for="u in kullananUrunler"
                :key="u.id"
                :to="{ name: 'urunler-id', params: { id: u.id } }"
                :class="{ 'text-disabled': !u.is_active }"
              >
                <VListItemTitle class="text-body-2">
                  {{ u.name }}
                </VListItemTitle>

                <template #append>
                  <VChip
                    v-if="!u.is_active"
                    size="x-small"
                    variant="tonal"
                  >
                    Pasif
                  </VChip>
                </template>
              </VListItem>
            </VList>
          </template>
        </VCard>
      </VCol>
    </VRow>

    <!-- 👉 Grup Oluştur / Düzenle -->
    <VDialog
      v-model="dialog"
      max-width="520"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>{{ editing ? 'Grubu Düzenle' : 'Yeni Grup' }}</VCardTitle>
        </VCardItem>

        <VCardText>
          <VForm
            ref="formRef"
            @submit.prevent="save"
          >
            <VRow>
              <VCol cols="12">
                <VTextField
                  v-model="form.name"
                  label="Grup Adı"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <VCol cols="12">
                <!--
                  Tip yalnızca oluştururken seçilir; sonradan değiştirilemez
                  çünkü mevcut seçenek değerleri (renk/metin) anlamsızlaşır.
                -->
                <VRadioGroup
                  v-if="!editing"
                  v-model="form.kind"
                  label="Tip"
                  inline
                >
                  <VRadio
                    label="Renk"
                    value="color"
                  />
                  <VRadio
                    label="Metin"
                    value="text"
                  />
                </VRadioGroup>

                <VTextField
                  v-else
                  :model-value="KIND_LABELS[form.kind]"
                  label="Tip"
                  readonly
                  disabled
                  hint="Tip değiştirilemez"
                  persistent-hint
                />
              </VCol>

              <VCol cols="12">
                <VSwitch
                  v-model="form.is_active"
                  label="Aktif"
                  hide-details
                />
              </VCol>
            </VRow>
          </VForm>
        </VCardText>

        <VCardActions class="px-6 pb-4">
          <VSpacer />
          <VBtn
            variant="text"
            :disabled="saving"
            @click="dialog = false"
          >
            Vazgeç
          </VBtn>
          <VBtn
            :loading="saving"
            @click="save"
          >
            Kaydet
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </div>
</template>
