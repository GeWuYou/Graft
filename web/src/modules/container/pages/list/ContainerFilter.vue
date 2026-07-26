<template>
  <section class="container-filter">
    <t-input
      class="container-filter__keyword"
      data-testid="container-filter-keyword"
      :value="modelValue.keyword"
      clearable
      :placeholder="t('container.list.filters.searchPlaceholder')"
      @change="update('keyword', String($event))"
      @enter="$emit('apply')"
      ><template #prefix-icon><search-icon /></template
    ></t-input>
    <t-button class="container-filter__trigger" variant="outline" @click="drawerVisible = true"
      ><template #icon><filter-icon /></template>{{ t('container.list.filters.more') }}</t-button
    >
    <div class="container-filter__desktop"><slot /></div>
    <t-drawer
      v-model:visible="drawerVisible"
      :header="t('container.list.filters.title')"
      placement="bottom"
      size="82%"
      :confirm-btn="t('container.list.filters.query')"
      :cancel-btn="t('container.list.filters.reset')"
      @confirm="applyFromDrawer"
      @cancel="$emit('reset')"
      ><div class="container-filter__drawer"><slot /></div
    ></t-drawer>
  </section>
</template>
<script setup lang="ts">
import { FilterIcon, SearchIcon } from 'tdesign-icons-vue-next';
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { ContainerFilters } from '../../types/container';
/** 筛选组件只管理移动端 Drawer 可见性，筛选值和查询时机仍由页面拥有。 */
const props = defineProps<{ modelValue: ContainerFilters }>();
const emit = defineEmits<{ 'update:modelValue': [value: ContainerFilters]; apply: []; reset: [] }>();
const { t } = useI18n();
const drawerVisible = ref(false);
function update<K extends keyof ContainerFilters>(key: K, value: ContainerFilters[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
}
function applyFromDrawer() {
  drawerVisible.value = false;
  emit('apply');
}
</script>
<style scoped lang="less">
.container-filter {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.container-filter > :first-child {
  flex: 1 1 18rem;
}

.container-filter__desktop {
  display: contents;
}

.container-filter__trigger {
  display: none;
}

.container-filter__drawer {
  display: grid;
  gap: var(--graft-density-gap-12);
}

@container (width < 768px) {
  .container-filter {
    width: 100%;
  }

  .container-filter__trigger {
    display: inline-flex;
  }

  .container-filter__desktop {
    display: none;
  }
}
</style>
