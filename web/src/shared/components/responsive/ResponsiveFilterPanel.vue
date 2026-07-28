<template>
  <section
    ref="container"
    :class="['responsive-filter-panel', `responsive-filter-panel--${variant.density}`]"
    :data-responsive-density="variant.density"
  >
    <div class="responsive-filter-panel__search"><slot name="search" /></div>
    <template v-if="variant.density === 'compact'">
      <t-tooltip :content="moreLabel" placement="top">
        <t-button
          :aria-label="moreLabel"
          class="responsive-filter-panel__trigger"
          shape="square"
          theme="default"
          variant="outline"
          @click="panelVisible = true"
        >
          <template #icon><filter-icon /></template>
        </t-button>
      </t-tooltip>
      <responsive-dialog v-model:visible="panelVisible" purpose="form" size="compact" :title="panelTitle">
        <div class="responsive-filter-panel__dialog-content"><slot name="filters" /></div>
      </responsive-dialog>
    </template>
    <div v-else class="responsive-filter-panel__filters"><slot name="filters" /></div>
  </section>
</template>
<script setup lang="ts">
import { FilterIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';

import { useResponsiveVariant, useViewportResponsiveVariant } from '@/shared/composables';

import ResponsiveDialog from './ResponsiveDialog.vue';

/** 筛选面板只决定筛选控件的呈现位置，筛选值和业务动作继续由调用页面拥有。 */
const props = withDefaults(
  defineProps<{
    densityScope?: 'container' | 'viewport';
    moreLabel: string;
    panelTitle: string;
  }>(),
  {
    densityScope: 'container',
  },
);

const container = ref<HTMLElement | null>(null);
const panelVisible = ref(false);
const containerVariant = useResponsiveVariant(container);
const viewportVariant = useViewportResponsiveVariant();
const variant = computed(() => (props.densityScope === 'viewport' ? viewportVariant.value : containerVariant.value));
</script>
<style scoped lang="less">
.responsive-filter-panel,
.responsive-filter-panel__search,
.responsive-filter-panel__filters,
.responsive-filter-panel__dialog-content {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.responsive-filter-panel {
  container-type: inline-size;
  flex: 1 1 auto;
}

.responsive-filter-panel__search {
  flex: 1 1 min-content;
}

.responsive-filter-panel__filters {
  flex: 1 1 auto;
}

.responsive-filter-panel__trigger {
  flex: 0 0 auto;
}

.responsive-filter-panel__dialog-content {
  align-items: stretch;
  flex-direction: column;
}

.responsive-filter-panel__dialog-content :deep(.t-select),
.responsive-filter-panel__dialog-content :deep(.t-button) {
  width: 100%;
}
</style>
