<script setup lang="ts">
/**
 * Sticky WhatsApp FAB — her sayfada duran genel iletişim yolu.
 *
 * Ürüne özel sipariş butonu ayrı (WhatsAppButton.vue, ürün detayda). Bu genel
 * olan; ürün bağlamı olmayan sayfalarda da müşteri tek tıkla ulaşabilsin.
 */
const link = useGeneralWhatsAppLink()
</script>

<template>
  <a
    :href="link"
    target="_blank"
    rel="noopener noreferrer"
    class="fixed bottom-20 right-5 z-40 flex size-14 items-center justify-center rounded-full bg-whatsapp text-white shadow-[0_4px_20px_rgba(0,0,0,0.15)] transition-transform duration-200 hover:scale-105 hover:bg-whatsapp-dark lg:bottom-8 lg:right-8"
    aria-label="WhatsApp ile iletişime geçin"
  >
    <IconWhatsApp class="size-7" />
  </a>
</template>
