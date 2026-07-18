<script setup lang="ts">
/**
 * Sepet drawer. Faz 2'de gerçek hale geldi — luxe redesign spec'i §2.1
 * "bu ekranlar atılacak değil, backend'e bağlanacak" demişti.
 *
 * DESIGN.md §Elevation: yalnızca drawer/modal gölge kullanabilir (%2 opacity).
 */
import { formatPrice } from '~/utils/price'

const acik = defineModel<boolean>({ required: true })

const { items, itemsTotal, remove, setQuantity } = useCart()

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
            class="flex size-10 items-center justify-center rounded-full border border-outline-variant/50 bg-surface-container-low text-primary transition-colors hover:bg-surface-container"
            aria-label="Kapat"
            @click="acik = false"
          >
            <Icon name="material-symbols:close" size="24" />
          </button>
        </header>

        <div v-if="!items.length" class="flex flex-1 flex-col items-center justify-center px-8 text-center">
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

        <!-- Dolu sepet -->
        <div v-else class="flex flex-1 flex-col overflow-hidden">
          <ul class="flex-1 overflow-y-auto px-6 py-4">
            <li
              v-for="item in items"
              :key="item.product_id"
              class="flex gap-4 border-b border-outline-variant/30 py-4"
            >
              <img
                v-if="item.image"
                :src="item.image"
                :alt="item.name"
                width="80"
                height="80"
                class="size-20 shrink-0 rounded-md object-cover"
              >
              <div
                v-else
                class="flex size-20 shrink-0 items-center justify-center rounded-md bg-surface-container-low"
              >
                <Icon name="material-symbols:local-florist-outline" size="24" class="text-outline-variant" />
              </div>

              <div class="min-w-0 flex-1">
                <NuxtLink
                  :to="`/urun/${item.slug}`"
                  class="line-clamp-2 text-body-md text-primary hover:underline"
                  @click="acik = false"
                >
                  {{ item.name }}
                </NuxtLink>
                <p class="mt-1 text-body-md font-medium text-primary">
                  {{ formatPrice(item.price) }}
                </p>

                <div class="mt-2 flex items-center gap-3">
                  <div class="flex items-center rounded border border-outline-variant/50">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-on-surface-variant hover:text-primary"
                      :aria-label="`${item.name} adedini azalt`"
                      @click="setQuantity(item.product_id, item.quantity - 1)"
                    >
                      −
                    </button>
                    <span class="min-w-8 text-center text-body-md">{{ item.quantity }}</span>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-on-surface-variant hover:text-primary"
                      :aria-label="`${item.name} adedini artır`"
                      @click="setQuantity(item.product_id, item.quantity + 1)"
                    >
                      +
                    </button>
                  </div>

                  <button
                    type="button"
                    class="text-xs text-on-surface-variant underline-offset-4 hover:text-primary hover:underline"
                    @click="remove(item.product_id)"
                  >
                    Kaldır
                  </button>
                </div>
              </div>
            </li>
          </ul>

          <div class="border-t border-outline-variant/30 px-6 py-5">
            <div class="flex items-center justify-between">
              <span class="text-body-md text-on-surface-variant">Ara Toplam</span>
              <span class="font-serif text-xl text-primary">{{ formatPrice(itemsTotal) }}</span>
            </div>
            <p class="mt-1 text-xs text-on-surface-variant">
              Teslimat ücreti sipariş adımında eklenir.
            </p>

            <NuxtLink
              to="/siparis"
              class="btn-primary text-label-caps mt-5 flex w-full items-center justify-center"
              @click="acik = false"
            >
              Siparişi Tamamla
            </NuxtLink>
          </div>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>
