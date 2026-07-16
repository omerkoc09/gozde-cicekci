<script setup lang="ts">
import { VForm } from 'vuetify/components'
import authV1BottomShape from '@images/svg/auth-v1-bottom-shape.svg'
import authV1TopShape from '@images/svg/auth-v1-top-shape.svg'
import { emailValidator, requiredValidator } from '@validators'
import ApiService from '@/services/ApiService'
import { ErrorPopup } from '@/utils/Popup'
import { useUserStore } from '@/store/user'

const loading = ref(false)
const isPasswordVisible = ref(false)
const formRef = ref<VForm>()

const form = ref({
  email: '',
  password: '',
})

const router = useRouter()

const onSubmit = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return
  console.log(import.meta.env.VITE_API_BASE_URL)
  loading.value = true
  const [error, resp] = await ApiService.post<any>('auth/login', form.value)
  loading.value = false
  if (error)
    return ErrorPopup(error)
  await useUserStore().login(resp.data.access_token, resp.data.refresh_token)

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

        <VCardText class="pt-1">
          <!--          <h5 class="text-h5 font-weight-semibold mb-1"> -->
          <!--            {{ themeConfig.app.title }}e Hoş Geldiniz! 👋🏻 -->
          <!--          </h5> -->
          <!--          <p class="mb-0"> -->
          <!--            Please sign-in to your account and start the adventure -->
          <!--          </p> -->
        </VCardText>

        <VCardText>
          <VForm
            ref="formRef"
            @submit.prevent="onSubmit"
          >
            <VRow>
              <!-- email -->
              <VCol cols="12">
                <VTextField
                  v-model="form.email"
                  label="Email"
                  type="email"
                  :rules="[requiredValidator, emailValidator]"
                />
              </VCol>

              <!-- password -->
              <VCol cols="12">
                <VTextField
                  v-model="form.password"
                  label="Parola"
                  :type="isPasswordVisible ? 'text' : 'password'"
                  :append-inner-icon="isPasswordVisible ? 'tabler-eye-off' : 'tabler-eye'"
                  :rules="[requiredValidator]"
                  @click:append-inner="isPasswordVisible = !isPasswordVisible"
                />

                <!-- remember me checkbox -->
                <div class="d-flex align-center justify-space-between flex-wrap mt-2 mb-4">
                  <!--                  <VCheckbox -->
                  <!--                    v-model="form.remember" -->
                  <!--                    label="Remember me" -->
                  <!--                  /> -->

                  <!--                  <RouterLink -->
                  <!--                    class="text-primary ms-2 mb-1" -->
                  <!--                    :to="{ name: 'auth-forgot-password' }" -->
                  <!--                  > -->
                  <!--                    Parolamı Unuttum? -->
                  <!--                  </RouterLink> -->
                </div>

                <!-- login button -->
                <VBtn
                  block
                  type="submit"
                  :loading="loading"
                >
                  GİRİŞ
                </VBtn>
              </VCol>

              <!-- create account -->
              <!--              <VCol -->
              <!--                cols="12" -->
              <!--                class="text-center text-base" -->
              <!--              > -->
              <!--                <span>Hesabınız yok mu?</span> -->
              <!--                <RouterLink -->
              <!--                  class="text-primary ms-2" -->
              <!--                  :to="{ name: 'auth-register' }" -->
              <!--                > -->
              <!--                  Hesap oluştur -->
              <!--                </RouterLink> -->
              <!--              </VCol> -->

              <!--
                <VCol
                cols="12"
                class="d-flex align-center"
                >
                <VDivider />
                <span class="mx-4">or</span>
                <VDivider />
                </VCol>

                &lt;!&ndash; auth providers &ndash;&gt;
                <VCol
                cols="12"
                class="text-center"
                >
                <AuthProvider />
                </VCol>
              -->
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
