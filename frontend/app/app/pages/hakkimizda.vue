<script setup lang="ts">
useSeoMeta({
  title: 'Hakkımızda | Çiçekçi',
  description: 'Taze çiçek ve buketlerle özel günlerinize dokunuyoruz.',
})
</script>

<template>
  <div class="kapsayici bolum yazi">
    <h1>Hakkımızda</h1>

    <p>
      Her buket bir mesaj taşır: "seni düşünüyorum", "tebrikler",
      "geçmiş olsun". Biz o mesajı taze çiçeklerle iletiyoruz.
    </p>

    <p>
      Mevsimin en güzel çiçeklerini seçiyor, siparişlerinizi özenle
      hazırlıyoruz. Beğendiğiniz ürünü seçin, WhatsApp'tan tek mesajla
      sipariş verin — gerisini biz hallederiz.
    </p>

    <NuxtLink
      to="/urunler"
      class="buton"
    >
      Ürünleri Gör
    </NuxtLink>
  </div>
</template>

<style scoped>
.yazi {
  max-inline-size: 60ch;
}

.yazi p {
  margin-block-end: 1rem;
}

.yazi .buton {
  margin-block-start: 1rem;
}
</style>
