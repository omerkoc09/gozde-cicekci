<script setup lang="ts">
const whatsappLink = useGeneralWhatsAppLink()

useSeoMeta({
  title: 'İletişim | Gözde Tasarım Çiçekçilik',
  description: 'Bize ulaşın: telefon, adres ve WhatsApp. Nişantaşı atölyemizden size özel çiçek tasarımları.',
  ogTitle: 'İletişim | Gözde Tasarım Çiçekçilik',
  ogType: 'website',
})
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <BreadCrumb :items="[{ label: 'Anasayfa', to: '/' }, { label: 'İletişim' }]" />

    <div class="mt-8 grid gap-14 md:grid-cols-2 md:gap-20">
      <div>
        <h1 class="font-serif text-4xl text-primary md:text-5xl">
          Bize Ulaşın
        </h1>
        <p class="mt-4 max-w-md text-body-md text-on-surface-variant">
          Siparişiniz, özel tasarım talepleriniz veya merak ettikleriniz için
          bize yazın. En kısa sürede dönüş yapıyoruz.
        </p>

        <div class="mt-10">
          <ContactDetails />
        </div>
      </div>

      <!-- WhatsApp kartı: sitenin tek gerçek dönüşüm yolu (spec §2.3),
           bu sayfada en görünür yerde durmalı. -->
      <div class="rounded-lg border border-outline-variant/40 bg-surface-container-low p-8 md:p-10">
        <span class="flex size-12 items-center justify-center rounded-full bg-whatsapp/10 text-whatsapp-dark">
          <IconWhatsApp class="size-6" />
        </span>

        <h2 class="mt-6 font-serif text-2xl text-primary">
          En hızlısı WhatsApp
        </h2>
        <p class="mt-3 text-body-md text-on-surface-variant">
          Siparişlerinizi WhatsApp üzerinden alıyoruz. Beğendiğiniz tasarımı
          gönderin, teslimat detaylarını birlikte belirleyelim.
        </p>

        <a
          :href="whatsappLink"
          target="_blank"
          rel="noopener noreferrer"
          class="text-label-caps mt-8 flex min-h-12 w-full items-center justify-center gap-2.5 rounded bg-whatsapp text-white transition-colors hover:bg-whatsapp-dark"
        >
          <IconWhatsApp class="size-5 shrink-0" />
          WhatsApp'tan Yazın
        </a>
      </div>
    </div>
  </div>
</template>
