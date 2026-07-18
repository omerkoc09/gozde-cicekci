<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import { useAdminUsers } from '@/composables/useAdminUsers'
import type { AdminUser } from '@/model/adminUser'
import { useUserStore } from '@/store/user'
import { ConfirmPopup, ErrorPopup, SuccessToast } from '@/utils/Popup'
import { confirmedValidator, requiredValidator } from '@validators'

// Backend kuralıyla tutarlı: min 8 karakter (kompleksite şartı yok).
const minLengthValidator = (value: string) =>
  value.length >= 8 || 'Şifre en az 8 karakter olmalı'

const api = useAdminUsers()
const userStore = useUserStore()

const loading = ref(false)
const users = ref<AdminUser[]>([])

const headers = [
  { title: 'Kullanıcı Adı', key: 'username' },
  { title: 'İşlemler', key: 'actions', sortable: false, align: 'end' as const, width: 150 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  users.value = data ?? []
}

onMounted(load)

// 👉 Yeni admin ekle
const createDialog = ref(false)
const creating = ref(false)
const createFormRef = ref<VForm>()

const createForm = ref({
  username: '',
  password: '',
  passwordConfirm: '',
})

const openCreate = () => {
  createForm.value = { username: '', password: '', passwordConfirm: '' }
  createDialog.value = true
}

const create = async () => {
  const { valid } = await createFormRef.value!.validate()
  if (!valid)
    return

  creating.value = true

  const [err] = await api.create({
    username: createForm.value.username,
    password: createForm.value.password,
  })

  creating.value = false

  if (err)
    return ErrorPopup(err.message)

  createDialog.value = false
  SuccessToast('Admin oluşturuldu')
  await load()
}

// 👉 Şifre sıfırla
const passwordDialog = ref(false)
const resettingPassword = ref(false)
const passwordFormRef = ref<VForm>()
const passwordTarget = ref<AdminUser | null>(null)

const passwordForm = ref({
  password: '',
  passwordConfirm: '',
})

const openResetPassword = (user: AdminUser) => {
  passwordTarget.value = user
  passwordForm.value = { password: '', passwordConfirm: '' }
  passwordDialog.value = true
}

const resetPassword = async () => {
  const { valid } = await passwordFormRef.value!.validate()
  if (!valid)
    return

  resettingPassword.value = true

  const [err] = await api.resetPassword(passwordTarget.value!.id, passwordForm.value.password)

  resettingPassword.value = false

  if (err)
    return ErrorPopup(err.message)

  passwordDialog.value = false
  SuccessToast('Şifre güncellendi')
}

// 👉 Sil
// Bir admin kendi hesabını silemez — backend de reddeder, burada önden engellenir.
const isSelf = (user: AdminUser) => user.username === userStore.username

const remove = async (user: AdminUser) => {
  const ok = await ConfirmPopup(
    `"${user.username}" kullanıcısı silinecek. Devam edilsin mi?`,
    'Sil',
    'Vazgeç',
  )

  if (!ok)
    return

  const [err] = await api.remove(user.id)
  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Kullanıcı silindi')
  await load()
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Kullanıcılar
      </h4>

      <VBtn
        prepend-icon="tabler-plus"
        @click="openCreate"
      >
        Yeni Yönetici
      </VBtn>
    </div>

    <VCard>
      <VDataTable
        :headers="headers"
        :items="users"
        :loading="loading"
        items-per-page="-1"
        no-data-text="Kayıtlı admin yok"
        loading-text="Yükleniyor..."
      >
        <template #item.actions="{ item }">
          <VBtn
            icon="tabler-key"
            variant="text"
            size="small"
            title="Şifre sıfırla"
            @click="openResetPassword(item)"
          />
          <VTooltip
            :disabled="!isSelf(item)"
            text="Kendi hesabınızı silemezsiniz"
            location="top"
          >
            <template #activator="{ props }">
              <span v-bind="props">
                <VBtn
                  icon="tabler-trash"
                  variant="text"
                  size="small"
                  color="error"
                  :disabled="isSelf(item)"
                  @click="remove(item)"
                />
              </span>
            </template>
          </VTooltip>
        </template>
      </VDataTable>
    </VCard>

    <!-- 👉 Yeni Admin Ekle -->
    <VDialog
      v-model="createDialog"
      max-width="480"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>Yeni Admin Ekle</VCardTitle>
        </VCardItem>

        <VCardText>
          <VForm
            ref="createFormRef"
            @submit.prevent="create"
          >
            <VRow>
              <VCol cols="12">
                <VTextField
                  v-model="createForm.username"
                  label="Kullanıcı Adı"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <VCol cols="12">
                <VTextField
                  v-model="createForm.password"
                  label="Şifre"
                  type="password"
                  :rules="[requiredValidator, minLengthValidator]"
                />
              </VCol>

              <VCol cols="12">
                <VTextField
                  v-model="createForm.passwordConfirm"
                  label="Şifre (Tekrar)"
                  type="password"
                  :rules="[requiredValidator, v => confirmedValidator(v, createForm.password)]"
                />
              </VCol>
            </VRow>
          </VForm>
        </VCardText>

        <VCardActions class="px-6 pb-4">
          <VSpacer />
          <VBtn
            variant="text"
            :disabled="creating"
            @click="createDialog = false"
          >
            Vazgeç
          </VBtn>
          <VBtn
            :loading="creating"
            @click="create"
          >
            Kaydet
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- 👉 Şifre Sıfırla -->
    <VDialog
      v-model="passwordDialog"
      max-width="480"
      persistent
    >
      <VCard>
        <VCardItem>
          <VCardTitle>"{{ passwordTarget?.username }}" Şifresini Sıfırla</VCardTitle>
        </VCardItem>

        <VCardText>
          <VForm
            ref="passwordFormRef"
            @submit.prevent="resetPassword"
          >
            <VRow>
              <VCol cols="12">
                <VTextField
                  v-model="passwordForm.password"
                  label="Yeni Şifre"
                  type="password"
                  :rules="[requiredValidator, minLengthValidator]"
                />
              </VCol>

              <VCol cols="12">
                <VTextField
                  v-model="passwordForm.passwordConfirm"
                  label="Yeni Şifre (Tekrar)"
                  type="password"
                  :rules="[requiredValidator, v => confirmedValidator(v, passwordForm.password)]"
                />
              </VCol>
            </VRow>
          </VForm>
        </VCardText>

        <VCardActions class="px-6 pb-4">
          <VSpacer />
          <VBtn
            variant="text"
            :disabled="resettingPassword"
            @click="passwordDialog = false"
          >
            Vazgeç
          </VBtn>
          <VBtn
            :loading="resettingPassword"
            @click="resetPassword"
          >
            Kaydet
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </div>
</template>
