<script setup lang="ts">
/**
 * Mobil alt navigasyon — referans mockup'taki 4'lü yapı (Mağaza/Koleksiyonlar/
 * Sepet/Hesabım). Sadece mobil/tablette görünür (lg:hidden); masaüstünde
 * header zaten aynı kısayolları içeriyor.
 *
 * "Favoriler" sekmesi kaldırıldı — favoriler/adresler backend'de yok ve
 * mock ekranları silindi (üyelik spec'i 2026-08-13). Yerine ürün listesine
 * giden "Koleksiyonlar" kondu, 4'lü yapı korunuyor.
 */
defineEmits<{ openCart: [] }>()

const { count: cartCount } = useCart()
const route = useRoute()

function aktifMi(to: string) {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}
</script>

<template>
  <nav
    class="fixed inset-x-0 bottom-0 z-40 flex items-stretch border-t border-outline-variant/30 bg-surface/95 backdrop-blur-md lg:hidden"
    style="padding-bottom: env(safe-area-inset-bottom)"
    aria-label="Alt gezinme"
  >
    <NuxtLink
      to="/"
      class="flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[11px] transition-colors"
      :class="aktifMi('/') ? 'text-primary' : 'text-on-surface-variant'"
    >
      <Icon :name="aktifMi('/') ? 'material-symbols:storefront' : 'material-symbols:storefront-outline'" size="22" />
      Mağaza
    </NuxtLink>

    <NuxtLink
      to="/urunler"
      class="flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[11px] transition-colors"
      :class="aktifMi('/urunler') ? 'text-primary' : 'text-on-surface-variant'"
    >
      <Icon :name="aktifMi('/urunler') ? 'material-symbols:local-florist' : 'material-symbols:local-florist-outline'" size="22" />
      Koleksiyonlar
    </NuxtLink>

    <button
      type="button"
      class="relative flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[11px] text-on-surface-variant transition-colors"
      @click="$emit('openCart')"
    >
      <span class="relative">
        <Icon name="material-symbols:shopping-cart-outline" size="22" />
        <span
          v-if="cartCount > 0"
          class="absolute -right-2 -top-1.5 flex size-4 items-center justify-center rounded-full bg-secondary text-[10px] font-medium text-white"
        >
          {{ cartCount > 9 ? '9+' : cartCount }}
        </span>
      </span>
      Sepet
    </button>

    <NuxtLink
      to="/hesabim"
      class="flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[11px] transition-colors"
      :class="aktifMi('/hesabim') ? 'text-primary' : 'text-on-surface-variant'"
    >
      <Icon :name="aktifMi('/hesabim') ? 'material-symbols:person' : 'material-symbols:person-outline'" size="22" />
      Hesabım
    </NuxtLink>
  </nav>
</template>
