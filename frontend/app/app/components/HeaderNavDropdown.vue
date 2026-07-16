<script setup lang="ts">
import type { Category } from '~/types/api'

/**
 * Nav dropdown — kategori eksenlerini açar (spec §5).
 * Hover ile açılır (masaüstü alışkanlığı) ama klavye/tıklama da çalışır;
 * hover'a bağlı kalmak klavye kullanıcısını dışarıda bırakırdı.
 */
defineProps<{
  label: string
  items: Category[]
  open: boolean
}>()

const emit = defineEmits<{ toggle: [], close: [] }>()

const kok = ref<HTMLElement | null>(null)

// Dışarı tıklayınca kapat
function disaTikla(e: MouseEvent) {
  if (kok.value && !kok.value.contains(e.target as Node))
    emit('close')
}

onMounted(() => document.addEventListener('click', disaTikla))
onBeforeUnmount(() => document.removeEventListener('click', disaTikla))
</script>

<template>
  <div
    ref="kok"
    class="relative"
    @mouseenter="!open && emit('toggle')"
    @mouseleave="emit('close')"
  >
    <button
      type="button"
      class="text-nav-link flex items-center gap-1 py-1 text-on-surface-variant transition-colors duration-300 hover:text-secondary"
      :aria-expanded="open"
      aria-haspopup="true"
      @click="emit('toggle')"
    >
      {{ label }}
      <Icon
        name="material-symbols:keyboard-arrow-down"
        size="16"
        class="transition-transform duration-200"
        :class="{ 'rotate-180': open }"
      />
    </button>

    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 translate-y-1"
      leave-active-class="transition duration-100 ease-in"
      leave-to-class="opacity-0 translate-y-1"
    >
      <!-- pt-3: buton ile menü arasındaki boşluk hover'ı kesmesin -->
      <div v-if="open" class="absolute left-1/2 top-full z-50 -translate-x-1/2 pt-3">
        <div
          class="min-w-52 rounded-lg border border-outline-variant/40 bg-surface-container-lowest py-2 shadow-[0_8px_32px_rgba(0,0,0,0.06)]"
        >
          <NuxtLink
            v-for="item in items"
            :key="item.id"
            :to="`/kategori/${item.slug}`"
            class="block px-5 py-2.5 text-body-md text-on-surface-variant transition-colors hover:bg-surface-container-low hover:text-primary"
          >
            {{ item.name }}
          </NuxtLink>
        </div>
      </div>
    </Transition>
  </div>
</template>
