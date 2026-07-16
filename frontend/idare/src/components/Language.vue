<script setup lang="ts">
defineSlots<{
  lang(props: { lang: string }): any
}>()

const selectedLanguage = ref<string>('en')
</script>

<template>
  <VCard
    flat
    width="100%"
    density="compact"
  >
    <VTabs v-model="selectedLanguage">
      <VTab value="en">
        English
      </VTab>
      <VTab value="tr">
        Turkish
      </VTab>
    </VTabs>

    <VWindow>
      <VWindowItem>
        <VRow
          :key="selectedLanguage + Math.random()"
          class="mt-4"
        >
          <slot
            name="lang"
            :lang="selectedLanguage"
          />
        </VRow>
      </VWindowItem>
    </VWindow>
  </VCard>
</template>
