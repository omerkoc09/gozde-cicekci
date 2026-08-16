<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import { useCategories } from '@/composables/useCategories'
import type { Axis, Category } from '@/model/category'
import { AXIS_LABELS } from '@/model/category'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'
import { requiredValidator } from '@validators'

const api = useCategories()

const loading = ref(false)
const categories = ref<Category[]>([])

const dialog = ref(false)
const saving = ref(false)
const formRef = ref<VForm>()

// Düzenlenen kategori; null ise yeni kayıt.
const editing = ref<Category | null>(null)

// sort_order YOK: sıra formdan değil, listedeki ok butonlarından değişir.
const form = ref({
  name: '',
  axis: 'occasion' as Axis,
  is_active: true,
  is_featured: false,
})

// Kart görseli. Boş bırakılırsa mevcut görsel korunur; yeni kategoride
// görsel zorunlu değil — site yedek görsele düşer.
const imageFile = ref<File[]>([])

const secilenDosya = computed(() => imageFile.value[0])

const preview = ref('')

watch(secilenDosya, file => {
  if (preview.value)
    URL.revokeObjectURL(preview.value)

  preview.value = file ? URL.createObjectURL(file) : ''
})

onBeforeUnmount(() => {
  if (preview.value)
    URL.revokeObjectURL(preview.value)
})

const byAxis = (axis: Axis) =>
  categories.value
    .filter(c => c.axis === axis)
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'tr'))

const occasionCategories = computed(() => byAxis('occasion'))
const typeCategories = computed(() => byAxis('type'))

// Kolon sıralaması kapalı: ok butonları satır index'ine dayanıyor, tablo
// yeniden sıralanırsa yanlış çift takas edilir. TÜM kolonlarda sortable:
// false olmalı — tek bir sıralanabilir başlık bile move()'u bozar.
const headers = [
  { title: 'Görsel', key: 'image', sortable: false, width: 90 },
  { title: 'Ad', key: 'name', sortable: false },
  { title: 'Aktif', key: 'is_active', sortable: false, width: 110 },
  { title: 'Öne Çıkan', key: 'is_featured', sortable: false, width: 130 },
  { title: 'Sıra', key: 'sira', sortable: false, width: 110 },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 150 },
]

const arama = ref('')

const siralaniyor = ref(false)

/**
 * Kategoriyi bir sıra kaydırır. Backend eksenin TÜM id'lerini istiyor;
 * gönderilen liste ekrandaki (filtresiz) sıranın tamamı.
 *
 * Arama doluyken kilitli: kullanıcı filtrelenmiş listede komşu görünen iki
 * satırı takas ettiğini sanır ama gerçek listede aralarında başka
 * kategoriler vardır — sonuç şaşırtıcı olur.
 */
const move = async (axis: Axis, index: number, direction: -1 | 1) => {
  const liste = byAxis(axis)
  const next = index + direction
  if (next < 0 || next >= liste.length)
    return

  const ids = liste.map(c => c.id)

  ;[ids[index], ids[next]] = [ids[next], ids[index]]

  siralaniyor.value = true

  const [err, guncel] = await api.reorder(axis, ids)

  siralaniyor.value = false

  if (err)
    return ErrorPopup(err.message)

  // Uç güncel listenin tamamını dönüyor — yeniden yüklemeye gerek yok.
  categories.value = guncel ?? []
}

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  categories.value = data ?? []
}

onMounted(load)

const openCreate = () => {
  editing.value = null
  form.value = { name: '', axis: 'occasion', is_active: true, is_featured: false }
  imageFile.value = []
  dialog.value = true
}

const openEdit = (c: Category) => {
  editing.value = c
  form.value = {
    name: c.name,
    axis: c.axis,
    is_active: c.is_active,
    is_featured: c.is_featured,
  }
  imageFile.value = []
  dialog.value = true
}

const save = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  saving.value = true

  // axis yalnızca oluştururken gönderilir — değiştirilemez (spec §4.1).
  const [err, kategori] = editing.value
    ? await api.update(editing.value.id, {
      name: form.value.name,
      is_active: form.value.is_active,
      is_featured: form.value.is_featured,
    })
    : await api.create({ ...form.value })

  if (err) {
    saving.value = false

    return ErrorPopup(err.message)
  }

  // Görsel ayrı uçta: yeni kategoride önce kayıt oluşur, id'siyle yüklenir.
  // Dosya seçilmediyse hiç dokunulmaz — düzenlemede mevcut görsel korunur.
  if (secilenDosya.value) {
    const [imgErr] = await api.replaceImage(kategori.id, secilenDosya.value)
    if (imgErr) {
      saving.value = false
      await load() // kategori kaydedildi, görsel yüklenemedi — liste doğruyu göstersin

      return ErrorPopup(`Kategori kaydedildi ama görsel yüklenemedi: ${imgErr.message}`)
    }
  }

  saving.value = false
  dialog.value = false
  SuccessToast(editing.value ? 'Kategori güncellendi' : 'Kategori oluşturuldu')
  await load()
}

const removeImage = async (c: Category) => {
  const ok = await ConfirmPopup(
    `"${c.name}" kategorisinin görseli kaldırılacak. Site varsayılan görseli gösterecek. Devam edilsin mi?`,
    'Kaldır',
    'Vazgeç',
  )

  if (!ok)
    return

  const [err] = await api.removeImage(c.id)
  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Görsel kaldırıldı')
  await load()
}

/**
 * Switch'ler anında kaydeder. İyimser değil: yanıt gelene kadar satırdaki
 * değer eski kalır, hata olursa hiç değişmez.
 */
const toggle = async (c: Category, field: 'is_active' | 'is_featured', value: boolean) => {
  const [err] = await api.update(c.id, { [field]: value })
  if (err)
    return ErrorPopup(err.message)

  await load()
}

const remove = async (c: Category) => {
  const [err, data] = await api.productCount(c.id)
  if (err)
    return ErrorPopup(err.message)

  const n = data.product_count

  const ok = await ConfirmPopup(
    n > 0
      ? `"${c.name}" kategorisinde ${n} ürün var. Silerseniz bu ürünler kategoriden çıkacak (ürünler silinmez). Devam edilsin mi?`
      : `"${c.name}" kategorisi silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [delErr] = await api.remove(c.id)
  if (delErr)
    return ErrorPopup(delErr.message)

  SuccessToast('Kategori silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Kategoriler
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="openCreate"
      >
        Yeni Kategori
      </VBtn>
    </div>

    <VAlert
      type="info"
      variant="tonal"
      class="mb-6"
    >
      Pasif kategori ana sayfada görünmez — öne çıkarılmış olsa bile.
      Kategori linki (slug) oluşturulduktan sonra değişmez.
      Sitedeki sırayı değiştirmek için satırdaki yukarı/aşağı oklarını kullanın.
    </VAlert>

    <VTextField
      v-model="arama"
      prepend-inner-icon="tabler-search"
      placeholder="Kategori ara..."
      class="mb-6"
      style="max-width: 360px;"
      hide-details
      clearable
    />

    <VCard
      v-for="axis in (['occasion', 'type'] as Axis[])"
      :key="axis"
      class="mb-6"
    >
      <VCardItem>
        <VCardTitle>{{ AXIS_LABELS[axis] }}</VCardTitle>
      </VCardItem>

      <VDataTable
        :headers="headers"
        :items="axis === 'occasion' ? occasionCategories : typeCategories"
        :search="arama"
        :loading="loading"
        items-per-page="-1"
        no-data-text="Bu eksende bu aramaya uyan kategori yok"
        loading-text="Yükleniyor..."
      >
        <template #item.image="{ item }">
          <VImg
            v-if="item.url_400"
            :src="item.url_400"
            :alt="item.name"
            width="48"
            height="64"
            cover
            class="rounded my-2"
          />
          <!--
            Görsel yoksa site yedek görsel gösteriyor; panelde bunu
            açıkça belirt ki "yüklendi mi?" diye tereddüt olmasın.
          -->
          <VTooltip
            v-else
            text="Görsel yok — site varsayılan görseli gösterir"
            location="top"
          >
            <template #activator="{ props }">
              <VIcon
                v-bind="props"
                icon="tabler-photo-off"
                size="20"
                class="text-disabled my-2"
              />
            </template>
          </VTooltip>
        </template>

        <template #item.name="{ item }">
          <span :class="{ 'text-disabled': !item.is_active }">{{ item.name }}</span>
        </template>

        <template #item.is_active="{ item }">
          <VSwitch
            :model-value="item.is_active"
            density="compact"
            hide-details
            @update:model-value="toggle(item, 'is_active', $event as boolean)"
          />
        </template>

        <template #item.is_featured="{ item }">
          <VTooltip
            :disabled="item.is_active"
            text="Pasif kategori öne çıkarılamaz"
            location="top"
          >
            <template #activator="{ props }">
              <div
                v-bind="props"
                class="d-inline-block"
              >
                <VSwitch
                  :model-value="item.is_featured"
                  :disabled="!item.is_active"
                  density="compact"
                  hide-details
                  @update:model-value="toggle(item, 'is_featured', $event as boolean)"
                />
              </div>
            </template>
          </VTooltip>
        </template>

        <!--
          Sıra artık sayı olarak girilmiyor: esnaf ok butonlarıyla taşıyor,
          sunucu sort_order'ı kendisi hesaplıyor. index sıralı listedeki
          konum — VDataTable items'ı byAxis() ile aynı sırada aldığı için
          move() de aynı diziyi kullanabiliyor.
        -->
        <template #item.sira="{ index }">
          <VBtn
            icon="tabler-chevron-up"
            variant="text"
            size="x-small"
            :disabled="siralaniyor || !!arama || index === 0"
            :title="arama ? 'Sıralamak için aramayı temizleyin' : 'Yukarı taşı'"
            @click="move(axis, index, -1)"
          />
          <VBtn
            icon="tabler-chevron-down"
            variant="text"
            size="x-small"
            :disabled="siralaniyor || !!arama
              || index === (axis === 'occasion' ? occasionCategories : typeCategories).length - 1"
            :title="arama ? 'Sıralamak için aramayı temizleyin' : 'Aşağı taşı'"
            @click="move(axis, index, 1)"
          />
        </template>

        <template #item.actions="{ item }">
          <VBtn
            v-if="item.url_400"
            icon="tabler-photo-off"
            variant="text"
            size="small"
            title="Görseli kaldır"
            @click="removeImage(item)"
          />
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
            @click="remove(item)"
          />
        </template>
      </VDataTable>
    </VCard>

    <!-- 👉 Oluştur / Düzenle -->
    <VDialog
      v-model="dialog"
      max-width="520"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>{{ editing ? 'Kategoriyi Düzenle' : 'Yeni Kategori' }}</VCardTitle>
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
                  label="Kategori Adı"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <VCol cols="12">
                <!--
                  Eksen yalnızca oluştururken seçilir; sonradan
                  değiştirilemez çünkü ürün ilişkileri anlamsızlaşır.
                -->
                <VRadioGroup
                  v-if="!editing"
                  v-model="form.axis"
                  label="Eksen"
                  inline
                >
                  <VRadio
                    label="Gönderim Amacı"
                    value="occasion"
                  />
                  <VRadio
                    label="Ürün Tipi"
                    value="type"
                  />
                </VRadioGroup>

                <VTextField
                  v-else
                  :model-value="AXIS_LABELS[form.axis]"
                  label="Eksen"
                  readonly
                  disabled
                  hint="Eksen değiştirilemez"
                  persistent-hint
                />
              </VCol>

              <VCol cols="12">
                <VFileInput
                  v-model="imageFile"
                  label="Kart Görseli"
                  accept="image/jpeg,image/png,image/webp"
                  prepend-icon=""
                  prepend-inner-icon="tabler-photo"
                  :hint="editing?.url_400
                    ? 'Boş bırakılırsa mevcut görsel korunur'
                    : 'İsteğe bağlı — yüklenmezse site varsayılan görseli gösterir. Dikey (3:4), en az 900px genişlik önerilir.'"
                  persistent-hint
                />
              </VCol>

              <!-- Önizleme: yeni dosya seçildiyse onu, seçilmediyse mevcudu. -->
              <VCol
                v-if="preview || editing?.url_400"
                cols="12"
              >
                <VImg
                  :src="preview || editing?.url_900"
                  :aspect-ratio="3 / 4"
                  cover
                  max-width="180"
                  class="rounded border"
                />
              </VCol>

              <VCol cols="12">
                <VSwitch
                  v-model="form.is_active"
                  label="Aktif"
                  hide-details
                />
                <VSwitch
                  v-model="form.is_featured"
                  label="Öne Çıkan"
                  :disabled="!form.is_active"
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
