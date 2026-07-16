import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-07-16',
  devtools: { enabled: true },

  modules: ['@nuxt/image', '@nuxtjs/sitemap', '@nuxt/fonts', '@nuxt/icon'],

  css: ['~/assets/css/main.css'],

  // Tailwind 4 — config JS dosyasında değil, main.css içinde @theme ile.
  vite: {
    plugins: [tailwindcss()],
  },

  // Referans tasarım Google Fonts CDN'den çekiyordu; self-host ediyoruz —
  // ek DNS+TLS round-trip LCP'yi geciktirir (spec §3.2).
  fonts: {
    families: [
      { name: 'Libre Caslon Text', provider: 'google', weights: [400, 700], styles: ['normal', 'italic'] },
      { name: 'Work Sans', provider: 'google', weights: [300, 400, 500, 600], styles: ['normal'] },
    ],
  },

  // Referans Material Symbols ikon fontunu CDN'den çekiyordu (~2MB, 32 ikon).
  // Iconify koleksiyonundan yalnızca kullanılanlar bundle'a girer (spec §3.2).
  icon: {
    mode: 'svg',
    serverBundle: { collections: ['material-symbols'] },
  },

  runtimeConfig: {
    // Proxy'nin arkasındaki gerçek Go API — yalnızca sunucuda (Nitro) görünür,
    // tarayıcıya sızmaz. NUXT_API_BASE env var'ıyla override edilir.
    goApiBase: process.env.NUXT_API_BASE || 'http://localhost:8080/api',

    public: {
      // Composable'ların çağırdığı adres — same-origin Nitro proxy'si.
      // Hem SSR hem client çağrıları buraya gider, CORS'a takılmaz.
      apiBase: '/api/go',

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
