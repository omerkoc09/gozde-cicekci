<script setup lang="ts">
import type { VForm } from 'vuetify/lib/components/VForm/index.mjs'
import authV1BottomShape from '@images/svg/auth-v1-bottom-shape.svg'
import authV1TopShape from '@images/svg/auth-v1-top-shape.svg'
import { requiredValidator } from '@validators'
import { ErrorPopup } from '@/utils/Popup'
import { useUserStore } from '@/store/user'

const loading = ref(false)
const isPasswordVisible = ref(false)
const formRef = ref<VForm>()

const form = ref({
  username: '',
  password: '',
})

const router = useRouter()

const onSubmit = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  loading.value = true

  const errMsg = await useUserStore().login(form.value.username, form.value.password)

  loading.value = false

  if (errMsg)
    return ErrorPopup(errMsg)

  await router.push('/')
}
</script>

<template>
  <div class="auth-wrapper d-flex align-center justify-center pa-4">
    <div class="position-relative my-sm-16">
      <!-- 👉 Top shape -->
      <VImg
        :src="authV1TopShape"
        class="auth-v1-top-shape d-none d-sm-block"
      />

      <!-- 👉 Bottom shape -->
      <VImg
        :src="authV1BottomShape"
        class="auth-v1-bottom-shape d-none d-sm-block"
      />

      <!-- 👉 Auth Card -->
      <VCard
        class="auth-card pa-4"
        max-width="448"
        min-width="448"
      >
        <VCardItem class="justify-center">
          <template #prepend>
            <div class="d-flex">
              <img
                src="/logo.png"
                alt="Logo"
                style="height: 100px;"
              >
            </div>
          </template>
        </VCardItem>

        <VCardText>
          <VForm
            ref="formRef"
            @submit.prevent="onSubmit"
          >
            <VRow>
              <!-- kullanıcı adı -->
              <VCol cols="12">
                <VTextField
                  v-model="form.username"
                  label="Kullanıcı Adı"
                  autocomplete="username"
                  :rules="[requiredValidator]"
                />
              </VCol>

              <!-- parola -->
              <VCol cols="12">
                <VTextField
                  v-model="form.password"
                  label="Parola"
                  autocomplete="current-password"
                  :type="isPasswordVisible ? 'text' : 'password'"
                  :append-inner-icon="isPasswordVisible ? 'tabler-eye-off' : 'tabler-eye'"
                  :rules="[requiredValidator]"
                  class="mb-4"
                  @click:append-inner="isPasswordVisible = !isPasswordVisible"
                />

                <!-- giriş butonu -->
                <VBtn
                  block
                  type="submit"
                  :loading="loading"
                >
                  GİRİŞ
                </VBtn>
              </VCol>
            </VRow>
          </VForm>
        </VCardText>
      </VCard>
    </div>
  </div>
</template>

<style lang="scss">
@use "@core/scss/template/pages/page-auth.scss";
</style>

<route lang="yaml">
meta:
  layout: blank
  redirectIfLoggedIn: true
</route>
