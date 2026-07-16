// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-07-16',
  devtools: { enabled: true },

  modules: ['@nuxt/image', '@nuxtjs/sitemap'],

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      // SSR'da sunucu-sunucu çağrısı yapılır; tarayıcıdan çağrılırsa CORS'a takılır
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080/api',
      whatsappNumber: process.env.NUXT_PUBLIC_WHATSAPP_NUMBER || '905551234567',
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',

      // İletişim bilgileri (spec §5.3 — settings tablosu ertelendi, .env'den okunuyor)
      contactPhone: process.env.NUXT_PUBLIC_CONTACT_PHONE || '',
      contactAddress: process.env.NUXT_PUBLIC_CONTACT_ADDRESS || '',
      contactHours: process.env.NUXT_PUBLIC_CONTACT_HOURS || '',
    },
  },

  // Görseller backend'den geliyor (local'de :8080, prod'da R2 CDN)
  image: {
    domains: ['localhost'],
  },

  site: {
    url: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',
    name: 'Çiçekçi',
  },

  sitemap: {
    sources: ['/api/_sitemap-urls'],
  },

  app: {
    head: {
      htmlAttrs: { lang: 'tr' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },
})
