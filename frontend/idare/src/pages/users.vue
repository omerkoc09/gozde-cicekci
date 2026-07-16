<script setup lang="ts">
import tarihFormat from '@/utils/ExDate'

import Extable from '@/components/extable.vue'
import type { ITableColumn } from '@/model/table'
import { emailValidator, phoneValidator, requiredValidator } from '@validators'
import { ApiQuery } from '@/model/api'

export interface User {
  id: number
  name: string
  surname: string
  phone: string
  email: string
  password: string
  created_at: string
  role: string
}

const apiUrl = 'user/'

const form = ref<User>({
  id: 0,
  name: '',
  surname: '',
  phone: '',
  email: '',
  password: '',
  created_at: '',
  role: '',
})

const tableQuery = ref<ApiQuery>(new ApiQuery())

const columns = ref<ITableColumn[]>([
  {
    key: 'name',
    name: 'İSİM',
    sortable: true,
  },
  {
    key: 'surname',
    name: 'SOYİSİM',
    sortable: true,
  },
  {
    key: 'phone',
    name: 'TELEFON',
  },
  {
    key: 'created_at',
    name: 'KAYIT TARİHİ',
  },
])

const roles = ref([
  { value: 1, title: 'Normal' },
  { value: 10, title: 'Admin' },
])
</script>

<template>
  <div>
    <VCard>
      <VCardText>
        <VRow>
          <VCol
            sm="6"
            md="4"
            lg="3"
          >
            <VTextField
              label="Arama"
              @update:model-value="tableQuery.append($event, 'name;surname')"
            />
          </VCol>
        </VRow>
      </VCardText>
      <VDivider />
      <Extable
        v-model:form="form"
        :api-url="apiUrl"
        :columns="columns"
        :query="tableQuery"
        create-button
        table-actions
      >
        <template #created_at="{ row }">
          {{ tarihFormat(row.created_at) }}
        </template>

        <!-- 👉 Modal -->
        <template #modalBody="{ closemodal, formloading, iscreateform }">
          <VRow>
            <!-- Name -->
            <VCol
              cols="12"
              md="6"
            >
              <VTextField
                v-model="form.name"
                label="İsim"
                :rules="[requiredValidator]"
              />
            </VCol>
            <!-- Surname -->
            <VCol
              cols="12"
              md="6"
            >
              <VTextField
                v-model="form.surname"
                label="Soyisim"
                :rules="[requiredValidator]"
              />
            </VCol>
            <!-- email -->
            <VCol
              cols="12"
              md="6"
            >
              <VTextField
                v-model="form.email"
                label="Email"
                type="email"
                :rules="[requiredValidator, emailValidator]"
              />
            </VCol>

            <!-- phone -->
            <VCol
              cols="12"
              md="6"
            >
              <VTextField
                v-model="form.phone"
                label="Telefon"
                type="phone"
                :rules="[requiredValidator, phoneValidator]"
              />
            </VCol>

            <!-- password -->
            <VCol
              cols="12"
              md="6"
            >
              <VTextField
                v-model="form.password"
                label="Parola"
                type="text"
                :rules="iscreateform ? [requiredValidator] : [] "
              />
            </VCol>

            <!-- 👉 Select Role -->
            <VCol
              cols="12"
              md="6"
            >
              <VSelect
                v-model="form.role"
                label="Rol"
                :items="roles"
                :rules="[requiredValidator]"
                clear-icon="tabler-x"
              />
            </VCol>
            <VCol
              cols="12"
              class="d-flex flex-wrap justify-center gap-4"
            >
              <VBtn
                type="submit"
                :loading="formloading"
              >
                Kaydet
              </VBtn>

              <VBtn
                type="reset"
                color="secondary"
                variant="tonal"
                :loading="formloading"
                @click="closemodal"
              >
                Vazgeç
              </VBtn>
            </VCol>
          </VRow>
        </template>
      </Extable>
    </vcard>
  </div>
</template>

<route lang="yaml">
meta:
  role: admin
</route>
