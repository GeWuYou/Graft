<template>
  <management-toolbar>
    <template #filters>
      <div class="query-filter-builder">
        <div class="query-filter-builder__top-row">
          <t-input
            :model-value="keyword"
            class="query-filter-builder__keyword management-query-search"
            clearable
            :placeholder="keywordPlaceholder"
            :title="keyword || keywordPlaceholder"
            @enter="$emit('search')"
            @update:model-value="$emit('update:keyword', normalizeTextValue($event))"
          />
          <slot name="toolbar-after-search" />
          <template v-if="!effectiveCompactMode">
            <slot name="saved-query-views" />
            <div class="query-filter-builder__actions">
              <t-button theme="primary" :loading="loading" @click="$emit('search')">
                {{ searchLabel }}
              </t-button>
              <t-button theme="default" variant="outline" @click="$emit('reset')">
                {{ resetLabel }}
              </t-button>
            </div>
          </template>
          <t-button
            v-else
            class="query-filter-builder__compact-search"
            theme="primary"
            block
            :loading="loading"
            @click="$emit('search')"
          >
            {{ searchLabel }}
          </t-button>
        </div>

        <t-collapse
          v-if="availableFields.length"
          v-model="filterPanels"
          borderless
          :class="[
            'query-filter-builder__filters-collapse',
            { 'query-filter-builder__filters-collapse--compact': effectiveCompactMode },
          ]"
        >
          <t-collapse-panel value="filters">
            <template v-if="effectiveCompactMode" #header>
              <span data-testid="query-filter-builder-compact-toggle">{{ compactFilterSummary }}</span>
            </template>
            <div v-if="effectiveCompactMode" class="query-filter-builder__compact-actions">
              <slot name="saved-query-views" />
              <t-button theme="default" variant="outline" block @click="$emit('reset')">
                {{ resetLabel }}
              </t-button>
            </div>
            <section class="query-filter-builder__group">
              <div class="query-filter-builder__group-header">
                <span class="query-filter-builder__group-title">{{ filtersGroupLabel }}</span>
              </div>
              <div class="query-filter-builder__group-body">
                <t-popup
                  v-model:visible="builderVisible"
                  attach="body"
                  destroy-on-close
                  placement="bottom-left"
                  trigger="click"
                >
                  <template #content>
                    <div class="query-filter-builder__popup">
                      <div class="query-filter-builder__header">
                        <span class="query-filter-builder__title">{{ builderTitle }}</span>
                        <span class="query-filter-builder__hint">{{ builderHint }}</span>
                      </div>

                      <div class="query-filter-builder__field-list">
                        <button
                          v-for="definition in availableFields"
                          :key="definition.key"
                          :class="[
                            'query-filter-builder__field-button',
                            {
                              'query-filter-builder__field-button--active': selectedFieldKey === definition.key,
                              'query-filter-builder__field-button--disabled': definition.disabled,
                            },
                          ]"
                          :disabled="definition.disabled"
                          type="button"
                          @click="$emit('update:selectedFieldKey', definition.key)"
                        >
                          {{ definition.label }}
                        </button>
                      </div>

                      <div v-if="selectedField" class="query-filter-builder__editor">
                        <div class="query-filter-builder__editor-title">
                          {{ selectedField.label }}
                        </div>

                        <template v-if="selectedField.key === timeFieldKey">
                          <div class="query-filter-builder__time-list">
                            <div v-for="field in timeFields" :key="field.key" class="query-filter-builder__time-item">
                              <span class="query-filter-builder__time-label">{{ field.label }}</span>
                              <t-date-range-picker
                                :model-value="field.value"
                                allow-input
                                clearable
                                enable-time-picker
                                format="YYYY-MM-DD HH:mm:ss"
                                :placeholder="field.placeholder"
                                @update:model-value="
                                  $emit('update:time-field', { key: field.key, value: normalizeRange($event) })
                                "
                              />
                            </div>
                          </div>
                        </template>

                        <template v-else-if="selectedField.key === sortFieldKey">
                          <div class="query-filter-builder__sort-list">
                            <div
                              v-for="(sorter, index) in sorters"
                              :key="`sort-row-${index}`"
                              class="query-filter-builder__sort-row"
                            >
                              <t-select
                                :model-value="sorter.field"
                                clearable
                                :options="sortFieldOptionsByIndex[index] ?? []"
                                :placeholder="sortFieldPlaceholder"
                                @update:model-value="$emit('update:sort-field', { index, value: $event })"
                              />
                              <t-select
                                :model-value="sorter.direction ?? 'desc'"
                                :options="sortDirectionOptions"
                                :placeholder="sortDirectionPlaceholder"
                                @update:model-value="$emit('update:sort-direction', { index, value: $event })"
                              />
                              <div class="query-filter-builder__sort-actions">
                                <t-button
                                  :disabled="sortMoveUpDisabled?.[index] ?? false"
                                  variant="text"
                                  theme="default"
                                  size="small"
                                  @click="$emit('move-sorter-up', index)"
                                >
                                  {{ moveUpLabel }}
                                </t-button>
                                <t-button
                                  :disabled="sortMoveDownDisabled?.[index] ?? false"
                                  variant="text"
                                  theme="default"
                                  size="small"
                                  @click="$emit('move-sorter-down', index)"
                                >
                                  {{ moveDownLabel }}
                                </t-button>
                                <t-button
                                  variant="text"
                                  theme="default"
                                  size="small"
                                  @click="$emit('remove-sorter', index)"
                                >
                                  {{ removeSorterLabel }}
                                </t-button>
                              </div>
                            </div>
                            <t-button
                              :disabled="sortAddDisabled"
                              theme="default"
                              variant="outline"
                              size="small"
                              @click="$emit('add-sorter')"
                            >
                              {{ addSorterLabel }}
                            </t-button>
                          </div>
                        </template>

                        <template v-else-if="selectedField.kind === 'select'">
                          <t-select
                            :model-value="fieldValue(selectedField.key)"
                            clearable
                            :options="selectedField.options ?? []"
                            :placeholder="selectedField.placeholder"
                            @update:model-value="
                              $emit('update:field', { key: selectedField.key, value: normalizeSelectValue($event) })
                            "
                          />
                        </template>

                        <template v-else-if="selectedField.kind === 'multi-select'">
                          <t-select
                            :model-value="fieldValue(selectedField.key)"
                            clearable
                            filterable
                            multiple
                            :min-collapsed-num="2"
                            :options="selectedField.options ?? []"
                            :placeholder="selectedField.placeholder"
                            @update:model-value="
                              $emit('update:field', { key: selectedField.key, value: normalizeArrayValue($event) })
                            "
                          />
                        </template>

                        <template v-else-if="selectedField.kind === 'tag-input'">
                          <t-tag-input
                            :model-value="tagInputValue(selectedField.key)"
                            clearable
                            :input-props="{ placeholder: selectedField.placeholder }"
                            @update:model-value="
                              $emit('update:field', { key: selectedField.key, value: normalizeArrayValue($event) })
                            "
                          />
                        </template>

                        <template v-else>
                          <t-input
                            :model-value="String(fieldValue(selectedField.key) ?? '')"
                            clearable
                            :placeholder="selectedField.placeholder"
                            @update:model-value="
                              $emit('update:field', { key: selectedField.key, value: normalizeTextValue($event) })
                            "
                          />
                        </template>
                      </div>
                    </div>
                  </template>

                  <t-button
                    :aria-expanded="builderVisible"
                    :theme="builderVisible ? 'primary' : 'default'"
                    :variant="builderVisible ? 'base' : 'dashed'"
                    >{{ addFilterLabel }}</t-button
                  >
                </t-popup>
              </div>
            </section>
          </t-collapse-panel>
        </t-collapse>

        <div v-if="presets.length" class="query-filter-builder__preset-row">
          <span class="query-filter-builder__preset-label">{{ presetLabel }}</span>
          <t-button
            v-for="preset in presets"
            :key="preset.key"
            size="small"
            :theme="activePreset === preset.key ? 'primary' : 'default'"
            :variant="activePreset === preset.key ? 'base' : 'outline'"
            @click="$emit('apply-preset', preset.key)"
          >
            {{ preset.title }}
          </t-button>
        </div>

        <div class="query-filter-builder__tag-row" data-testid="query-filter-builder-tags">
          <t-tag
            v-for="tag in tags"
            :key="tag.key"
            :closable="tag.closable !== false"
            max-width="280"
            size="small"
            theme="primary"
            variant="light-outline"
            @close="$emit('reset')"
          >
            {{ tag.label }}
          </t-tag>
        </div>
      </div>
    </template>
  </management-toolbar>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import { ManagementToolbar } from '@/shared/components/management';
import { useViewportResponsiveVariant } from '@/shared/composables/useViewportResponsiveVariant';

import type {
  AdvancedQueryFilterFieldDefinition,
  AdvancedQueryFilterPreset,
  AdvancedQueryFilterTag,
  AdvancedQuerySortItem,
  AdvancedQuerySortOption,
  AdvancedQueryTimeRangeField,
} from './query-filter-builder';

const props = withDefaults(
  defineProps<{
    activePreset: string;
    addFilterLabel: string;
    addSorterLabel: string;
    builderHint: string;
    builderTitle: string;
    compactMode?: boolean;
    compactToggleLabel?: string;
    fieldValues: Record<string, string | string[]>;
    fields: AdvancedQueryFilterFieldDefinition[];
    filtersGroupLabel: string;
    keyword: string;
    keywordPlaceholder: string;
    loading?: boolean;
    moveDownLabel: string;
    moveUpLabel: string;
    presetLabel: string;
    presets: AdvancedQueryFilterPreset[];
    removeSorterLabel: string;
    resetLabel: string;
    searchLabel: string;
    selectedFieldKey: string;
    sortAddDisabled?: boolean;
    sortDirectionOptions: AdvancedQuerySortOption[];
    sortDirectionPlaceholder: string;
    sortFieldOptionsByIndex: AdvancedQuerySortOption[][];
    sortFieldKey: string;
    sortFieldPlaceholder: string;
    sortMoveDownDisabled?: boolean[];
    sortMoveUpDisabled?: boolean[];
    sorters: AdvancedQuerySortItem[];
    showSorterBuilder?: boolean;
    tags: AdvancedQueryFilterTag[];
    timeFieldKey: string;
    timeFields: AdvancedQueryTimeRangeField[];
  }>(),
  {
    compactMode: false,
    compactToggleLabel: '',
    sortMoveDownDisabled: () => [],
    sortMoveUpDisabled: () => [],
  },
);

defineEmits<{
  (e: 'add-sorter'): void;
  (e: 'apply-preset', preset: string): void;
  (e: 'close-tag', key: string): void;
  (e: 'move-sorter-down', index: number): void;
  (e: 'move-sorter-up', index: number): void;
  (e: 'remove-sorter', index: number): void;
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:field', payload: { key: string; value: string | string[] }): void;
  (e: 'update:keyword', value: string): void;
  (e: 'update:selectedFieldKey', key: string): void;
  (
    e: 'update:sort-direction',
    payload: { index: number; value: string | number | Array<string | number> | undefined },
  ): void;
  (
    e: 'update:sort-field',
    payload: { index: number; value: string | number | Array<string | number> | undefined },
  ): void;
  (e: 'update:time-field', payload: { key: string; value: string[] }): void;
}>();

const builderVisible = ref(false);
const filterPanels = ref<string[]>(['filters']);
const viewportVariant = useViewportResponsiveVariant();
const effectiveCompactMode = computed(() => props.compactMode || viewportVariant.value.density === 'compact');
const compactFilterSummary = computed(() => {
  if (props.tags.length > 0) {
    return `${props.compactToggleLabel || props.filtersGroupLabel} (${props.tags.length})`;
  }

  return props.compactToggleLabel || props.filtersGroupLabel;
});

watch(
  effectiveCompactMode,
  (compactMode) => {
    filterPanels.value = compactMode ? [] : ['filters'];
  },
  { immediate: true },
);

const availableFields = computed(() =>
  props.fields.filter((field) => {
    if (field.key === props.timeFieldKey) {
      return props.timeFields.length > 0;
    }
    if (field.key === props.sortFieldKey) {
      return props.showSorterBuilder !== false;
    }
    return true;
  }),
);

const selectedField = computed(
  () => availableFields.value.find((field) => field.key === props.selectedFieldKey) ?? null,
);

function fieldValue(key: string) {
  return props.fieldValues[key] ?? '';
}

function tagInputValue(key: string) {
  const value = fieldValue(key);
  return Array.isArray(value) ? value : [];
}

function normalizeTextValue(value: string | number | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeSelectValue(value: string | number | Array<string | number> | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeArrayValue(value: string | number | Array<string | number> | undefined) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map((item) => String(item).trim()).filter(Boolean);
}

function normalizeRange(value: string[] | undefined) {
  return Array.isArray(value) ? value : [];
}
</script>
<style scoped lang="less">
.query-filter-builder {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-14);
  min-width: 0;
}

.query-filter-builder__top-row,
.query-filter-builder__group-body,
.query-filter-builder__preset-row,
.query-filter-builder__compact-actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.query-filter-builder__keyword {
  flex: 0 1 360px;
  max-width: 360px;
  min-width: 280px;
}

.query-filter-builder__actions {
  display: flex;
  gap: var(--graft-density-gap-12);
}

.query-filter-builder__compact-search {
  flex: 0 0 auto;
}

.query-filter-builder__filters-collapse {
  width: fit-content;
}

.query-filter-builder__filters-collapse:not(.query-filter-builder__filters-collapse--compact)
  :deep(.t-collapse-panel__header) {
  display: none;
}

.query-filter-builder__filters-collapse:not(.query-filter-builder__filters-collapse--compact)
  :deep(.t-collapse-panel__body),
.query-filter-builder__filters-collapse:not(.query-filter-builder__filters-collapse--compact)
  :deep(.t-collapse-panel__content) {
  padding: 0;
}

.query-filter-builder__compact-actions {
  align-items: center;
}

.query-filter-builder__group {
  padding: 0;
}

.query-filter-builder__group-header {
  display: none;
}

.query-filter-builder__group-title {
  display: none;
}

.query-filter-builder__popup {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(180px, 220px) minmax(320px, 420px);
  inline-size: min(660px, calc(100vw - 32px));
  min-inline-size: 560px;
  min-width: 0;
  padding: var(--graft-density-gap-8);
}

.query-filter-builder__header {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  grid-column: 1 / -1;
}

.query-filter-builder__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.query-filter-builder__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.query-filter-builder__field-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.query-filter-builder__field-button {
  background: var(--td-bg-color-container-hover);
  border: 1px solid transparent;
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  cursor: pointer;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.query-filter-builder__field-button--active {
  background: color-mix(in srgb, var(--td-brand-color-light) 40%, var(--td-bg-color-container) 60%);
  border-color: var(--td-brand-color);
}

.query-filter-builder__field-button--disabled {
  color: var(--td-text-color-disabled);
  cursor: not-allowed;
}

.query-filter-builder__editor {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.query-filter-builder__editor-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.query-filter-builder__time-list,
.query-filter-builder__sort-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.query-filter-builder__time-item {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.query-filter-builder__time-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.query-filter-builder__sort-row {
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(120px, 1fr) minmax(120px, 1fr);
}

.query-filter-builder__sort-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  grid-column: 1 / -1;
}

.query-filter-builder__preset-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.query-filter-builder__tag-row {
  align-items: center;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-8);
  min-height: 32px;
  overflow-x: auto;
  padding-bottom: var(--graft-density-gap-2);
}

.query-filter-builder__tag-row > :deep(.t-tag) {
  flex: 0 0 auto;
}

@media (width <= 900px) {
  .query-filter-builder__popup {
    grid-template-columns: 1fr;
    inline-size: min(420px, calc(100vw - 32px));
    min-inline-size: 0;
  }
}

@media (width <= 768px) {
  .query-filter-builder__top-row {
    align-items: stretch;
    flex-flow: column nowrap;
  }

  .query-filter-builder__keyword {
    flex: none;
    min-width: 0;
    width: 100%;
  }

  .query-filter-builder__compact-search {
    width: 100%;
  }

  .query-filter-builder__preset-row {
    flex-wrap: nowrap;
    max-width: 100%;
    overflow-x: auto;
    overscroll-behavior-inline: contain;
    padding-bottom: var(--graft-density-gap-4);
  }

  .query-filter-builder__preset-row :deep(.t-button) {
    flex: 0 0 auto;
  }
}
</style>
