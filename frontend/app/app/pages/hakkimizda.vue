<script setup lang="ts">
useSeoMeta({
  title: 'Hakkımızda | Gözde Tasarım Çiçekçilik',
  description: 'Sanatı doğayla buluşturan zarif tasarımlar. Mevsimin en taze çiçeklerini mimari bir bakışla aranje ediyoruz.',
  ogTitle: 'Hakkımızda | Gözde Tasarım Çiçekçilik',
  ogType: 'website',
})
</script>

<template>
  <div>
    <!-- Manifesto -->
    <section class="section-gap">
      <div class="site-container text-center">
        <img
          src="~/assets/img/gozde-icon.svg"
          alt=""
          aria-hidden="true"
          class="mx-auto h-24 w-auto md:h-28"
          width="82"
          height="120"
        >

        <h1 class="mt-10 font-serif text-4xl leading-tight text-primary md:text-5xl lg:text-6xl">
          Çiçeğin Sanatı.
        </h1>

        <p class="mx-auto mt-7 max-w-2xl text-body-md text-on-surface-variant md:text-body-lg">
          Gözde Tasarım Çiçekçilik olarak çiçeklerin birer süsten fazlası
          olduğuna inanıyoruz; onlar duyguların ifadesi, doğanın şiirinden
          donmuş anlardır. Atölyemiz mevsimin en seçkin çiçeklerini özenle
          seçmeye, onları mimari bir bakış ve organik formlarına derin bir
          saygıyla aranje etmeye adanmıştır.
        </p>
        <p class="mx-auto mt-5 max-w-2xl text-body-md text-on-surface-variant md:text-body-lg">
          Hayatın en anlamlı anları ve gündelik zarafetiniz için size özel
          çiçek deneyimleri tasarlıyoruz.
        </p>

        <div class="mt-12">
          <GoldDivider />
        </div>
      </div>
    </section>

    <!-- Atölye / iletişim -->
    <section class="bg-surface-container-low">
      <div class="site-container py-16 md:py-24">
        <div class="grid gap-12 md:grid-cols-2 md:items-center md:gap-16">
          <div>
            <h2 class="font-serif text-3xl text-primary md:text-4xl">
              Atölyemizi Ziyaret Edin
            </h2>
            <p class="mt-4 max-w-md text-body-md text-on-surface-variant">
              Çiçek sanatımızı yerinde deneyimlemeniz ya da size özel tasarımlar
              için bize ulaşmanız bizi mutlu eder.
            </p>

            <div class="mt-10">
              <ContactDetails />
            </div>
          </div>

          <!-- Referansta canlı Google Maps embed var; statik görsel + yol tarifi
               linki tercih edildi (spec §6.3): embed üçüncü taraf çerez taşır
               (KVKK) ve sayfayı yavaşlatır. -->
          <div class="rounded-lg border border-outline-variant/40 bg-surface p-10 text-center md:p-14">
            <Icon name="material-symbols:location-on-outline" size="34" class="text-secondary" />
            <p class="mt-4 font-serif text-xl text-primary">
              Ödemiş Atölyemiz
            </p>
            <p class="mx-auto mt-2 max-w-xs text-body-md text-on-surface-variant">
              Şht. Adnan Menderes Bulvarı üzerinde, ziyarete açığız.
            </p>
            <NuxtLink to="/iletisim" class="btn-secondary text-label-caps mt-8">
              İletişim Bilgileri
            </NuxtLink>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
