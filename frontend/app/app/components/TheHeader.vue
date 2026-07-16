<script setup lang="ts">
const acik = ref(false)
const route = useRoute()

// Sayfa değişince mobil menü kapansın
watch(() => route.fullPath, () => {
  acik.value = false
})
</script>

<template>
  <header class="ust">
    <div class="kapsayici ust-ic">
      <NuxtLink to="/" class="logo">
        Çiçekçi
      </NuxtLink>

      <button
        class="hamburger"
        :aria-expanded="acik"
        aria-label="Menü"
        @click="acik = !acik"
      >
        <span />
        <span />
        <span />
      </button>

      <nav
        class="nav"
        :class="{ 'nav-acik': acik }"
      >
        <NuxtLink to="/urunler">
          Ürünler
        </NuxtLink>
        <NuxtLink to="/hakkimizda">
          Hakkımızda
        </NuxtLink>
        <NuxtLink to="/iletisim">
          İletişim
        </NuxtLink>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.ust {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--renk-zemin);
  border-block-end: 1px solid var(--renk-cizgi);
}

.ust-ic {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  min-block-size: 60px;
}

.logo {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--renk-vurgu);
}

.hamburger {
  display: grid;
  gap: 5px;
  padding: 8px;
  border: 0;
  background: none;
  cursor: pointer;
}

.hamburger span {
  inline-size: 22px;
  block-size: 2px;
  background: var(--renk-metin);
}

/* Mobilde menü kapalı; hamburger açınca alta yayılıyor */
.nav {
  display: none;
  inline-size: 100%;
  flex-direction: column;
  padding-block-end: 0.75rem;
}

.nav-acik {
  display: flex;
}

.nav a {
  padding-block: 0.6rem;
  font-weight: 500;
}

.nav a:hover,
.nav a.router-link-active {
  color: var(--renk-vurgu);
}

@media (min-width: 768px) {
  .hamburger { display: none; }

  .nav {
    display: flex;
    flex-direction: row;
    gap: 1.5rem;
    inline-size: auto;
    padding-block-end: 0;
  }
}
</style>
