<script setup lang="ts">
import type { Product } from '~/types/api'

const props = defineProps<{ product: Product }>()

const link = useWhatsAppLink(() => props.product)
</script>

<template>
  <a
    :href="link"
    class="wa"
    target="_blank"
    rel="noopener"
  >
    <svg
      class="wa-ikon"
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="currentColor"
    >
      <path d="M17.5 14.4c-.3-.2-1.8-.9-2-1-.3-.1-.5-.2-.7.1-.2.3-.7 1-.9 1.2-.2.2-.3.2-.6.1-.3-.2-1.3-.5-2.4-1.5-.9-.8-1.5-1.8-1.7-2.1-.2-.3 0-.5.1-.6l.5-.5c.1-.2.2-.3.3-.5 0-.2 0-.4-.1-.5l-.9-2.2c-.2-.6-.5-.5-.7-.5h-.6c-.2 0-.5.1-.8.4-.3.3-1 1-1 2.4s1.1 2.8 1.2 3c.2.2 2.1 3.2 5.1 4.5.7.3 1.3.5 1.7.6.7.2 1.4.2 1.9.1.6-.1 1.8-.7 2-1.5.3-.7.3-1.3.2-1.5 0-.1-.2-.2-.5-.3z" />
      <path d="M12 2a10 10 0 0 0-8.6 15.1L2 22l5-1.3A10 10 0 1 0 12 2zm0 18.2c-1.5 0-3-.4-4.3-1.2l-.3-.2-3 .8.8-2.9-.2-.3A8.2 8.2 0 1 1 12 20.2z" />
    </svg>

    WhatsApp'tan Sipariş Ver
  </a>
</template>

<style scoped>
.wa {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  inline-size: 100%;
  padding: 0.95rem 1.5rem;
  border-radius: var(--yuvarlak);
  background: var(--renk-whatsapp);
  color: #fff;
  font-size: 1.05rem;
  font-weight: 700;
}

.wa:hover {
  background: var(--renk-whatsapp-koyu);
}

.wa-ikon {
  inline-size: 22px;
  block-size: 22px;
  flex-shrink: 0;
}
</style>
