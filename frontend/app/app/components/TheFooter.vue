<script setup lang="ts">
const { public: cfg } = useRuntimeConfig()
const yil = new Date().getFullYear()
</script>

<template>
  <footer class="alt">
    <div class="kapsayici alt-ic">
      <div>
        <strong>Çiçekçi</strong>
        <p
          v-if="cfg.contactAddress"
          class="soluk"
        >
          {{ cfg.contactAddress }}
        </p>
      </div>

      <div>
        <p v-if="cfg.contactPhone">
          <a :href="`tel:${cfg.contactPhone.replace(/\s/g, '')}`">{{ cfg.contactPhone }}</a>
        </p>
        <p
          v-if="cfg.contactHours"
          class="soluk"
        >
          {{ cfg.contactHours }}
        </p>
      </div>

      <nav class="alt-nav">
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

    <div class="kapsayici telif soluk">
      © {{ yil }} Çiçekçi
    </div>
  </footer>
</template>

<style scoped>
.alt {
  margin-block-start: 3rem;
  padding-block: 2rem 1rem;
  background: var(--renk-zemin-alt);
  border-block-start: 1px solid var(--renk-cizgi);
  font-size: 0.925rem;
}

.alt-ic {
  display: grid;
  gap: 1.5rem;
}

.alt p {
  margin: 0.25rem 0;
}

.alt-nav {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.alt-nav a:hover {
  color: var(--renk-vurgu);
}

.telif {
  margin-block-start: 1.5rem;
  padding-block-start: 1rem;
  border-block-start: 1px solid var(--renk-cizgi);
  font-size: 0.85rem;
}

@media (min-width: 640px) {
  .alt-ic { grid-template-columns: repeat(3, 1fr); }
}
</style>
