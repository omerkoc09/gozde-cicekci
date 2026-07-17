<script setup lang="ts">
import type { Axis, Category } from '~/types/api'
import { AXIS_LABELS } from '~/types/api'

/**
 * Kategori filtresi — DESIGN.md §Components: pill şeklinde chip'ler,
 * seçili olan dolu koyu, pasif olan outline.
 *
 * Filtre state'i URL'de (spec §5.6) — Vue state'inde değil. Böylece liste
 * paylaşılabiliyor ve tarayıcı geri tuşu çalışıyor.
 */
const { data: categories } = await useCategoryList()

const route = useRoute()
const router = useRouter()

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

// Ana sayfadan "Tümünü Gör" ile gelindiğinde ?eksen=occasion|type ile
// ilgili eksen grubuna scroll edilir — hiçbir spesifik kategori seçilmez,
// sadece kullanıcı doğru gruba yönlendirilir (spesifik axis filtresi
// backend'de yok, bkz. 2026-07-18 sipariş formu iyileştirmeleri kararı).
const hedefEksen = computed(() => route.query.eksen as Axis | undefined)

// Mobilde filtreler açılır panelde — yer kaplamasın. Hedef eksen varsa
// panel otomatik açılır, yoksa kapalı başlar.
const acik = ref(!!hedefEksen.value)

const eksenRefs: Partial<Record<Axis, HTMLElement>> = {}
function eksenRef(axis: Axis, el: Element | null) {
  if (el instanceof HTMLElement)
    eksenRefs[axis] = el
}

onMounted(() => {
  if (!hedefEksen.value)
    return

  eksenRefs[hedefEksen.value]?.scrollIntoView({ behavior: 'smooth', block: 'start' })
})

</script>

<template>
  <div>
    <!-- Mobil aç/kapa -->
    <button
      type="button"
      class="text-label-caps inline-flex items-center gap-2 rounded border border-outline-variant/60 px-4 py-2.5 text-on-surface md:hidden"
      :aria-expanded="acik"
      @click="acik = !acik"
    >
      <Icon name="material-symbols:tune" size="16" />
      Filtrele
      <span v-if="filtreVar" class="size-1.5 rounded-full bg-accent-gold" />
    </button>

    <div :class="acik ? 'mt-5 block' : 'hidden md:block'">
      <div
        v-for="axis in (['occasion', 'type'] as Axis[])"
        :key="axis"
        :ref="el => eksenRef(axis, el as Element | null)"
        class="mb-5 scroll-mt-24 last:mb-0"
      >
        <h3 class="text-label-caps mb-3 text-on-surface-variant/70">
          {{ AXIS_LABELS[axis] }}
        </h3>

        <div class="flex flex-wrap gap-2">
          <button
            v-for="kategori in (axis === 'occasion' ? occasionCategories : typeCategories)"
            :key="kategori.id"
            type="button"
            class="rounded-full border px-4 py-2 text-sm transition-colors"
            :class="seciliDeger(axis) === kategori.slug
              ? 'border-primary bg-primary text-on-primary'
              : 'border-outline-variant/60 text-on-surface-variant hover:border-primary hover:text-primary'"
            :aria-pressed="seciliDeger(axis) === kategori.slug"
            @click="sec(axis, kategori)"
          >
            {{ kategori.name }}
          </button>

          <p
            v-if="(axis === 'occasion' ? occasionCategories : typeCategories).length === 0"
            class="text-sm text-on-surface-variant/60"
          >
            —
          </p>
        </div>
      </div>

      <button
        v-if="filtreVar"
        type="button"
        class="text-label-caps mt-2 text-secondary hover:text-secondary-hover"
        @click="temizle"
      >
        Filtreyi Temizle
      </button>
    </div>
  </div>
</template>
