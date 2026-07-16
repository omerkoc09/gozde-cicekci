<script setup lang="ts">
/**
 * Sepet drawer — INERT (spec §2.1).
 *
 * Backend'de sepet yok. Bu drawer her zaman boş durumu gösterir ve müşteriyi
 * gerçek dönüşüm yoluna (WhatsApp / ürünler) yönlendirir. Sahte ürün listesi
 * göstermez — site sahip olmadığı bir durumu iddia etmemeli.
 *
 * DESIGN.md §Elevation: yalnızca drawer/modal gölge kullanabilir (%2 opacity).
 */
const acik = defineModel<boolean>({ required: true })

watch(acik, (a) => {
  if (import.meta.client)
    document.body.style.overflow = a ? 'hidden' : ''
})

onBeforeUnmount(() => {
  if (import.meta.client)
    document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <!-- Perde -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      leave-active-class="transition-opacity duration-150"
      leave-to-class="opacity-0"
    >
      <div
        v-if="acik"
        class="fixed inset-0 z-[60] bg-primary/20 backdrop-blur-[2px]"
        @click="acik = false"
      />
    </Transition>

    <!-- Panel -->
    <Transition
      enter-active-class="transition-transform duration-300 ease-out"
      enter-from-class="translate-x-full"
      leave-active-class="transition-transform duration-200 ease-in"
      leave-to-class="translate-x-full"
    >
      <aside
        v-if="acik"
        class="fixed inset-y-0 right-0 z-[70] flex w-full max-w-sm flex-col bg-surface shadow-[0_0_48px_rgba(0,0,0,0.08)]"
        role="dialog"
        aria-modal="true"
        aria-label="Sepet"
        @keydown.escape="acik = false"
      >
        <header class="flex items-center justify-between border-b border-outline-variant/30 px-6 py-5">
          <h2 class="font-serif text-xl text-primary">
            Sepetim
          </h2>
          <button
            type="button"
            class="rounded p-1.5 text-on-surface-variant transition-colors hover:text-primary"
            aria-label="Kapat"
            @click="acik = false"
          >
            <Icon name="material-symbols:close" size="22" />
          </button>
        </header>

        <div class="flex flex-1 flex-col items-center justify-center px-8 text-center">
          <Icon
            name="material-symbols:shopping-cart-outline"
            size="44"
            class="text-outline-variant"
          />
          <p class="mt-5 font-serif text-lg text-primary">
            Sepetiniz boş
          </p>
          <p class="mt-2 text-body-md text-on-surface-variant">
            Beğendiğiniz tasarımları keşfedin, siparişinizi WhatsApp üzerinden
            kolayca tamamlayın.
          </p>
          <NuxtLink to="/urunler" class="btn-primary text-label-caps mt-7" @click="acik = false">
            Koleksiyonu Keşfet
          </NuxtLink>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>
