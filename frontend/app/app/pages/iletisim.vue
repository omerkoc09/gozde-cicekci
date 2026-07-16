<script setup lang="ts">
const { public: cfg } = useRuntimeConfig()

const whatsappLink = useGeneralWhatsAppLink()

useSeoMeta({
  title: 'İletişim | Çiçekçi',
  description: 'Bize ulaşın: telefon, adres ve WhatsApp.',
})
</script>

<template>
  <div class="kapsayici bolum yazi">
    <h1>İletişim</h1>

    <ul class="bilgiler">
      <li v-if="cfg.contactPhone">
        <span class="etiket">Telefon</span>
        <a :href="`tel:${cfg.contactPhone.replace(/\s/g, '')}`">{{ cfg.contactPhone }}</a>
      </li>
      <li v-if="cfg.contactAddress">
        <span class="etiket">Adres</span>
        <span>{{ cfg.contactAddress }}</span>
      </li>
      <li v-if="cfg.contactHours">
        <span class="etiket">Çalışma Saatleri</span>
        <span>{{ cfg.contactHours }}</span>
      </li>
    </ul>

    <a
      :href="whatsappLink"
      class="wa"
      target="_blank"
      rel="noopener"
    >
      WhatsApp'tan Yazın
    </a>
  </div>
</template>

<style scoped>
.yazi {
  max-inline-size: 50ch;
}

.bilgiler {
  margin: 0 0 1.5rem;
  padding: 0;
  list-style: none;
}

.bilgiler li {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  padding-block: 0.75rem;
  border-block-end: 1px solid var(--renk-cizgi);
}

.etiket {
  font-size: 0.8rem;
  color: var(--renk-soluk);
}

.wa {
  display: inline-block;
  padding: 0.8rem 1.5rem;
  border-radius: var(--yuvarlak);
  background: var(--renk-whatsapp);
  color: #fff;
  font-weight: 700;
}

.wa:hover {
  background: var(--renk-whatsapp-koyu);
}
</style>
