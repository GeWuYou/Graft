<template>
  <div class="section preset-catalog">
    <div class="section-heading">
      <div class="section-title">{{ t('layout.setting.workbench.presets.title') }}</div>
      <div class="section-desc">{{ t('layout.setting.workbench.presets.description') }}</div>
    </div>

    <div class="preset-catalog__toolbar">
      <t-input
        v-model:value="keyword"
        class="preset-catalog__search"
        clearable
        size="small"
        type="search"
        :placeholder="t('layout.setting.workbench.presets.searchPlaceholder')"
        :aria-label="t('layout.setting.workbench.presets.searchPlaceholder')"
      />
      <t-radio-group
        class="preset-catalog__filters"
        variant="default-filled"
        theme="button"
        size="small"
        :value="activeCategory"
        :options="filterOptions"
        @change="handleCategoryChange"
      />
      <span class="preset-catalog__count">{{
        t('layout.setting.workbench.presets.resultCount', { count: visiblePresets.length })
      }}</span>
    </div>

    <div v-if="featuredPresets.length" class="preset-catalog__group">
      <div class="preset-catalog__group-title">{{ t('layout.setting.workbench.presets.featured') }}</div>
      <theme-workbench-preset-grid
        :presets="featuredPresets"
        :active-preset-id="activePresetId"
        @select="$emit('select', $event)"
      />
    </div>

    <div v-if="catalogPresets.length" class="preset-catalog__group">
      <div class="preset-catalog__group-title">
        {{
          t(featuredPresets.length ? 'layout.setting.workbench.presets.more' : 'layout.setting.workbench.presets.all')
        }}
      </div>
      <theme-workbench-preset-grid
        :presets="catalogPresets"
        :active-preset-id="activePresetId"
        @select="$emit('select', $event)"
      />
    </div>

    <div v-if="!visiblePresets.length" class="preset-catalog__empty">
      {{ t('layout.setting.workbench.presets.empty') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

import { t } from '@/locales';
import { useLocale } from '@/locales/useLocale';
import type { ThemePresetCategory, ThemePresetDefinition } from '@/types/theme';

import ThemeWorkbenchPresetGrid from './ThemeWorkbenchPresetGrid.vue';

type PresetFilter = 'all' | ThemePresetCategory;

/** 内置预设目录只负责本地筛选和展示，预设草稿的预览与保存仍由工作台 store 统一处理。 */
const props = defineProps<{
  presets: ThemePresetDefinition[];
  activePresetId: string | null;
}>();

defineEmits<{
  select: [presetId: string];
}>();

const { locale } = useLocale();
const keyword = ref('');
const activeCategory = ref<PresetFilter>('all');

const availableCategories = computed(() => {
  const categories = new Set(props.presets.map((preset) => preset.category));
  return [...categories] as ThemePresetCategory[];
});

const filterOptions = computed(() => [
  { value: 'all', label: t('layout.setting.workbench.presets.filters.all') },
  ...availableCategories.value.map((category) => ({
    value: category,
    label: t(`layout.setting.workbench.presets.categories.${category}`),
  })),
]);

const visiblePresets = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLocaleLowerCase(locale.value);

  return props.presets.filter((preset) => {
    if (activeCategory.value !== 'all' && preset.category !== activeCategory.value) {
      return false;
    }

    if (!normalizedKeyword) {
      return true;
    }

    const searchableText = [
      t(preset.labelKey),
      t(preset.descriptionKey),
      t(`layout.setting.workbench.presets.categories.${preset.category}`),
    ]
      .join(' ')
      .toLocaleLowerCase(locale.value);

    return searchableText.includes(normalizedKeyword);
  });
});

const hasActiveFilter = computed(() => activeCategory.value !== 'all' || Boolean(keyword.value.trim()));

const featuredPresets = computed(() => {
  if (hasActiveFilter.value) {
    return [];
  }

  return visiblePresets.value.filter((preset) => preset.featured).slice(0, 6);
});

const catalogPresets = computed(() => {
  if (!featuredPresets.value.length) {
    return visiblePresets.value;
  }

  const featuredIds = new Set(featuredPresets.value.map((preset) => preset.id));
  return visiblePresets.value.filter((preset) => !featuredIds.has(preset.id));
});

function handleCategoryChange(value: string | number | boolean) {
  activeCategory.value = value as PresetFilter;
}
</script>
<style lang="less" scoped>
.preset-catalog {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.preset-catalog__toolbar {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-auto-flow: dense;
  grid-template-columns: minmax(180px, 1fr) auto;
}

.preset-catalog__search {
  min-width: 0;
}

.preset-catalog__filters {
  grid-column: 1 / -1;
  min-width: 0;
}

.preset-catalog__filters :deep(.t-radio-group) {
  flex-wrap: wrap;
  max-width: 100%;
}

.preset-catalog__count {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  justify-self: end;
  white-space: nowrap;
}

.preset-catalog__group {
  display: grid;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.preset-catalog__group-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-weight: 700;
}

.preset-catalog__group :deep(.preset-grid) {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.preset-catalog__group :deep(.preset-card) {
  gap: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-10);
}

.preset-catalog__group :deep(.preset-card__thumb-shell) {
  min-height: 96px;
}

.preset-catalog__group :deep(.preset-card__thumbnail) {
  padding: var(--graft-density-gap-6);
}

.preset-catalog__group :deep(.preset-card__desc) {
  -webkit-box-orient: vertical;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-height: 1.45;
  overflow: hidden;
}

.preset-catalog__empty {
  align-items: center;
  background: var(--td-bg-color-page);
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-small);
  justify-content: center;
  min-height: 116px;
  padding: var(--graft-density-gap-16);
  text-align: center;
}

@media screen and (width <= 768px) {
  .preset-catalog__toolbar {
    grid-template-columns: minmax(0, 1fr);
  }

  .preset-catalog__count {
    justify-self: start;
  }

  .preset-catalog__group :deep(.preset-grid) {
    grid-template-columns: 1fr;
  }
}
</style>
