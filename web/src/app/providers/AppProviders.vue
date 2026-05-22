<template>
  <t-config-provider :global-config="getComponentsLocale">
    <div class="app-theme-surface" :class="mode" :data-theme-mode="mode">
      <router-view :key="locale" />
    </div>
    <setting-com />
  </t-config-provider>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import SettingCom from '@/layouts/setting.vue';
import { useLocale } from '@/locales/useLocale';
import { useSettingStore } from '@/store';

const store = useSettingStore();

const mode = computed(() => {
  return store.displayMode;
});

const { getComponentsLocale, locale } = useLocale();
</script>
<style lang="less" scoped>
.app-theme-surface {
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
  height: 100%;
  min-height: 100%;
}

#nprogress .bar {
  background: var(--td-brand-color) !important;
}
</style>
