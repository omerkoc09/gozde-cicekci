<template>
  <!-- Yükleniyor durumu — referansta yok (mockup'lar hep dolu veri gösteriyor)
       ama gerçek sitede liste yavaş gelebilir (spec §5.2). Ölçüler ProductCard
       ile birebir aynı: iskelet ile gerçek kart arasında sayfa zıplamasın. -->
  <div class="p-2" aria-hidden="true">
    <div
      class="mb-5 w-full animate-pulse rounded-md bg-surface-container"
      style="aspect-ratio: 4 / 5"
    />
    <div class="flex flex-col items-center gap-2 px-1 pb-2">
      <div class="h-4 w-3/4 animate-pulse rounded bg-surface-container" />
      <div class="h-3 w-1/3 animate-pulse rounded bg-surface-container" />
    </div>
  </div>
</template>
