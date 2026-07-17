<script setup lang="ts">
/**
 * Arama kutusu — ürün adı veya kategori adında arar (backend `q` parametresi).
 * Submit-driven: canlı arama/autocomplete yok, Enter'a basınca /urunler?q=...
 * adresine gider. Mevcut amac/tip filtreleri korunur (AND).
 */
const route = useRoute()
const router = useRouter()

const terim = ref((route.query.q as string) ?? '')

watch(() => route.query.q, (q) => {
  terim.value = (q as string) ?? ''
})

function ara() {
  const deger = terim.value.trim()
  const query: Record<string, string> = { ...route.query as Record<string, string> }

  if (deger)
    query.q = deger
  else
    delete query.q

  router.push({ path: '/urunler', query })
}
</script>

<template>
  <form class="flex items-center" role="search" @submit.prevent="ara">
    <div class="flex w-full items-center gap-2 rounded-full border border-outline-variant/40 bg-surface-container-low px-3.5 py-2">
      <Icon name="material-symbols:search" size="18" class="shrink-0 text-on-surface-variant" />
      <input
        v-model="terim"
        type="search"
        placeholder="Ürün veya kategori ara..."
        aria-label="Ürün veya kategori ara"
        class="w-full bg-transparent text-body-md text-primary placeholder:text-on-surface-variant/70 focus:outline-none"
      >
    </div>
  </form>
</template>
