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
  //
  // clientBundle.scan: kullanılan ikonları build sırasında tarayıp client
  // bundle'ına da gömer. Aksi halde dinamik isimli ikonlar (ör. bottom nav'daki
  // aktifMi() ? 'x' : 'y') ilk boyada boş gelip ancak Iconify API'sinden
  // yüklenince görünüyordu — "yenileyince geliyor" sorununun kök nedeni buydu.
  icon: {
    mode: 'svg',
    serverBundle: { collections: ['material-symbols'] },
    clientBundle: {
      scan: true,
      sizeLimitKb: 512,
    },
  },

  runtimeConfig: {
    // Proxy'nin arkasındaki gerçek Go API — yalnızca sunucuda (Nitro) görünür,
    // tarayıcıya sızmaz. Buradaki değer yalnızca DEV varsayılanı; production'da
    // RUNTIME'da NUXT_GO_API_BASE env var'ıyla override edilir (Nuxt runtimeConfig
    // anahtarları NUXT_<UPPER_SNAKE> ile eşlenir: goApiBase → NUXT_GO_API_BASE).
    // Not: burada process.env okumak build zamanında değerlendirilir ve default'u
    // imaja gömer — o yüzden okumuyoruz, override'ı Nuxt'un runtime mekanizmasına
    // bırakıyoruz.
    goApiBase: 'http://localhost:8080/api',

    public: {
      // Composable'ların çağırdığı adres — same-origin Nitro proxy'si.
      // Hem SSR hem client çağrıları buraya gider, CORS'a takılmaz.
      apiBase: '/api/go',

      whatsappNumber: process.env.NUXT_PUBLIC_WHATSAPP_NUMBER || '905052178501',
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',

      // İletişim bilgileri (spec §5.3 — settings tablosu ertelendi, .env'den
      // okunuyor). Varsayılanlar Gözde Tasarım'ın gerçek bilgileri; prod'da
      // env var ile override edilir.
      contactPhone: process.env.NUXT_PUBLIC_CONTACT_PHONE || '+90 553 614 00 63',
      contactAddress: process.env.NUXT_PUBLIC_CONTACT_ADDRESS || 'Atatürk, Şht. Adnan Menderes Blv. No:31/A, 35750 Ödemiş/İzmir',
      contactHours: process.env.NUXT_PUBLIC_CONTACT_HOURS || 'Her gün 09:00 - 22:00',
      contactMapsUrl: process.env.NUXT_PUBLIC_CONTACT_MAPS_URL || 'https://www.google.com/maps/place/G%C3%B6zde+Tasar%C4%B1m+%C3%87i%C3%A7ek%C3%A7ilik/@38.229775,27.9810084,17z/data=!3m1!4b1!4m6!3m5!1s0x14b8e36cbb50999d:0x6032dce6058bd479!8m2!3d38.2297708!4d27.9835833!16s%2Fg%2F11z94dr3m5',
      instagramUrl: process.env.NUXT_PUBLIC_INSTAGRAM_URL || 'https://www.instagram.com/gozde_tasarm_cicekcilik?igsh=MWEwbG1naGgycWtrOA==',
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
      link: [
        // Gerçek marka logosu — tarayıcı sekmesi ikonu. Hem SVG (modern
        // tarayıcı, keskin) hem .ico (gerçek logodan üretildi, evrensel
        // fallback) hem apple-touch (PNG) veriliyor — her ortamda doğru logo.
        { rel: 'icon', type: 'image/svg+xml', href: '/gozde-icon.svg' },
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico', sizes: '16x16 32x32 48x48' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
      ],
    },
  },
})
