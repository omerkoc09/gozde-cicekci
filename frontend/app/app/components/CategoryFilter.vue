<script setup lang="ts">
import type { Axis, Category } from '~/types/api'
import { AXIS_LABELS } from '~/types/api'

const { data: categories } = await useCategoryList()

const route = useRoute()
const router = useRouter()

// Filtre state'i URL'de (spec §5.6) — Vue state'inde değil.
// Böylece liste paylaşılabiliyor ve tarayıcı geri tuşu çalışıyor.
const amac = computed(() => route.query.amac as string | undefined)
const tip = computed(() => route.query.tip as string | undefined)

const byAxis = (axis: Axis) =>
  (categories.value ?? []).filter(c => c.axis === axis)

const occasionCategories = computed(() => byAxis('occasion'))
const typeCategories = computed(() => byAxis('type'))

const seciliDeger = (axis: Axis) => (axis === 'occasion' ? amac.value : tip.value)

// Seçim URL query'sini günceller; aynı slug'a tekrar tıklamak filtreyi kaldırır.
function sec(axis: Axis, category: Category) {
  const key = axis === 'occasion' ? 'amac' : 'tip'
  const mevcut = { ...route.query }

  if (mevcut[key] === category.slug)
    delete mevcut[key]
  else
    mevcut[key] = category.slug

  router.push({ query: mevcut })
}

const filtreVar = computed(() => !!amac.value || !!tip.value)

function temizle() {
  router.push({ query: {} })
}

// Mobilde filtreler açılır panelde — yer kaplamasın
const acik = ref(false)
</script>

<template>
  <div class="filtre">
    <button
      class="filtre-ac"
      :aria-expanded="acik"
      @click="acik = !acik"
    >
      Filtrele
      <span v-if="filtreVar" class="rozet" />
    </button>

    <div
      class="filtre-govde"
      :class="{ 'filtre-govde-acik': acik }"
    >
      <div
        v-for="axis in (['occasion', 'type'] as Axis[])"
        :key="axis"
        class="grup"
      >
        <h3 class="grup-baslik">
          {{ AXIS_LABELS[axis] }}
        </h3>

        <div class="cipler">
          <button
            v-for="kategori in (axis === 'occasion' ? occasionCategories : typeCategories)"
            :key="kategori.id"
            class="cip"
            :class="{ 'cip-secili': seciliDeger(axis) === kategori.slug }"
            @click="sec(axis, kategori)"
          >
            {{ kategori.name }}
          </button>

          <p
            v-if="(axis === 'occasion' ? occasionCategories : typeCategories).length === 0"
            class="soluk"
          >
            —
          </p>
        </div>
      </div>

      <button
        v-if="filtreVar"
        class="temizle"
        @click="temizle"
      >
        Filtreyi temizle
      </button>
    </div>
  </div>
</template>

<style scoped>
.filtre-ac {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.1rem;
  border: 1px solid var(--renk-cizgi);
  border-radius: var(--yuvarlak);
  background: var(--renk-zemin);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.rozet {
  inline-size: 8px;
  block-size: 8px;
  border-radius: 50%;
  background: var(--renk-vurgu);
}

.filtre-govde {
  display: none;
  margin-block-start: 1rem;
}

.filtre-govde-acik {
  display: block;
}

.grup {
  margin-block-end: 1.25rem;
}

.grup-baslik {
  margin: 0 0 0.6rem;
  font-size: 0.95rem;
}

.cipler {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.cip {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--renk-cizgi);
  border-radius: 999px;
  background: var(--renk-zemin);
  font: inherit;
  font-size: 0.9rem;
  cursor: pointer;
}

.cip:hover {
  border-color: var(--renk-vurgu);
}

.cip-secili {
  border-color: var(--renk-vurgu);
  background: var(--renk-vurgu);
  color: #fff;
}

.temizle {
  padding: 0;
  border: 0;
  background: none;
  color: var(--renk-vurgu);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

/* Masaüstünde filtre hep açık, buton gizli */
@media (min-width: 768px) {
  .filtre-ac { display: none; }

  .filtre-govde {
    display: block;
    margin-block-start: 0;
  }
}
</style>
