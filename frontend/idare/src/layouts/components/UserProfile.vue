<script setup lang="ts">
import placeholder from '@images/avatars/placeholder.png'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const router = useRouter()

const logout = async () => {
  await userStore.logout()
  await router.push({ name: 'auth-login' })
}
</script>

<template>
  <VBadge
    dot
    location="bottom right"
    offset-x="3"
    offset-y="3"
    bordered
    color="success"
  >
    <VAvatar
      class="cursor-pointer"
      color="primary"
      variant="tonal"
    >
      <VImg :src="placeholder" />

      <!-- SECTION Menu -->
      <VMenu
        activator="parent"
        width="230"
        location="bottom end"
        offset="14px"
      >
        <VList>
          <!-- 👉 Oturum açan kullanıcı -->
          <VListItem>
            <VListItemTitle class="font-weight-semibold">
              {{ userStore.username || 'Yönetici' }}
            </VListItemTitle>
          </VListItem>

          <VDivider class="my-2" />

          <!-- 👉 Logout -->
          <VListItem @click="logout">
            <template #prepend>
              <VIcon
                class="me-2"
                icon="tabler-logout"
                size="22"
              />
            </template>

            <VListItemTitle>Çıkış</VListItemTitle>
          </VListItem>
        </VList>
      </VMenu>
      <!-- !SECTION -->
    </VAvatar>
  </VBadge>
</template>
