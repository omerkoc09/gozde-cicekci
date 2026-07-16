<script setup lang="ts">
/**
 * İletişim bilgileri — referanstaki "Visit Our Atelier" bloğu.
 * Hem hakkımızda hem iletişim sayfasında kullanılıyor.
 *
 * Bilgiler env var'dan geliyor (spec §5.3 — settings tablosu ertelendi).
 * Boş olan alan hiç render edilmez, yer tutucu göstermez.
 */
const { public: cfg } = useRuntimeConfig()

/** Google Maps yol tarifi — canlı embed yerine link (spec §6.3: KVKK + hız). */
const haritaLinki = computed(() =>
  cfg.contactAddress
    ? `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(cfg.contactAddress)}`
    : null,
)
</script>

<template>
  <ul class="space-y-6">
    <li v-if="cfg.contactPhone" class="flex items-start gap-4">
      <span class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md border border-outline-variant/50 text-secondary">
        <Icon name="material-symbols:call-outline" size="17" />
      </span>
      <span>
        <span class="text-label-caps block text-on-surface-variant/70">Telefon</span>
        <a
          :href="`tel:${cfg.contactPhone.replace(/\s/g, '')}`"
          class="mt-1 block text-body-md text-on-surface hover:text-secondary"
        >
          {{ cfg.contactPhone }}
        </a>
      </span>
    </li>

    <li class="flex items-start gap-4">
      <span class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md border border-outline-variant/50 text-secondary">
        <Icon name="material-symbols:mail-outline" size="17" />
      </span>
      <span>
        <span class="text-label-caps block text-on-surface-variant/70">WhatsApp</span>
        <a
          :href="useGeneralWhatsAppLink()"
          target="_blank"
          rel="noopener noreferrer"
          class="mt-1 block text-body-md text-on-surface hover:text-secondary"
        >
          Mesaj gönderin
        </a>
      </span>
    </li>

    <li v-if="cfg.contactAddress" class="flex items-start gap-4">
      <span class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md border border-outline-variant/50 text-secondary">
        <Icon name="material-symbols:location-on-outline" size="17" />
      </span>
      <span>
        <span class="text-label-caps block text-on-surface-variant/70">Atölye</span>
        <span class="mt-1 block text-body-md text-on-surface">{{ cfg.contactAddress }}</span>
        <a
          v-if="haritaLinki"
          :href="haritaLinki"
          target="_blank"
          rel="noopener noreferrer"
          class="text-label-caps mt-2 inline-flex items-center gap-1 text-secondary hover:text-secondary-hover"
        >
          Yol Tarifi Al
          <Icon name="material-symbols:arrow-outward" size="13" />
        </a>
      </span>
    </li>

    <li v-if="cfg.contactHours" class="flex items-start gap-4">
      <span class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md border border-outline-variant/50 text-secondary">
        <Icon name="material-symbols:schedule-outline" size="17" />
      </span>
      <span>
        <span class="text-label-caps block text-on-surface-variant/70">Çalışma Saatleri</span>
        <span class="mt-1 block text-body-md text-on-surface">{{ cfg.contactHours }}</span>
      </span>
    </li>
  </ul>
</template>
