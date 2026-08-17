<script setup lang="ts">
import type { Product } from '~/types/api'

/**
 * Ürün detaydaki WhatsApp sipariş CTA'sı — sitenin TEK gerçek dönüşüm yolu
 * (spec §2.3). "Sepete Ekle" inert olduğu için bu buton kaybolmamalı; yoksa
 * site demo değil kırık olur.
 */
const props = withDefaults(defineProps<{
  product: Product

  /** Tükenen üründe mesaj "ne zaman gelir"e döner (spec §6.1). */
  outOfStock?: boolean
}>(), { outOfStock: false })

const link = useWhatsAppLink(() => props.product, () => props.outOfStock)
</script>

<template>
  <a
    :href="link"
    target="_blank"
    rel="noopener noreferrer"
    class="text-label-caps flex min-h-12 w-full items-center justify-center gap-2.5 rounded border border-whatsapp text-whatsapp-dark transition-colors hover:bg-whatsapp hover:text-white"
  >
    <IconWhatsApp class="size-5 shrink-0" />
    {{ outOfStock ? 'WhatsApp\'tan Sor' : 'WhatsApp\'tan Sipariş Ver' }}
  </a>
</template>
