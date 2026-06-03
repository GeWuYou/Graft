<template>
  <management-toolbar>
    <template #filters>
      <div class="log-filter-builder">
        <div class="log-filter-builder__top-row">
          <t-input
            :model-value="keyword"
            class="log-filter-builder__keyword"
            clearable
            :placeholder="keywordPlaceholder"
            @update:model-value="$emit('update:keyword', normalizeTextValue($event))"
          />
          <div class="log-filter-builder__actions">
            <t-button theme="primary" :loading="loading" @click="$emit('search')">
              {{ searchLabel }}
            </t-button>
            <t-button theme="default" variant="outline" @click="$emit('reset')">
              {{ resetLabel }}
            </t-button>
          </div>
        </div>

        <section class="log-filter-builder__group">
          <div class="log-filter-builder__group-header">
            <span class="log-filter-builder__group-title">{{ filtersGroupLabel }}</span>
          </div>
          <div class="log-filter-builder__group-body">
            <t-popup
              v-model:visible="builderVisible"
              attach="body"
              destroy-on-close
              placement="bottom-left"
              trigger="click"
            >
              <template #content>
                <div class="log-filter-builder__popup">
                  <div class="log-filter-builder__header">
                    <span class="log-filter-builder__title">{{ builderTitle }}</span>
                    <span class="log-filter-builder__hint">{{ builderHint }}</span>
                  </div>

                  <div class="log-filter-builder__field-list">
                    <button
                      v-for="definition in fields"
                      :key="definition.key"
                      :class="[
                        'log-filter-builder__field-button',
                        {
                          'log-filter-builder__field-button--active': selectedFieldKey === definition.key,
                          'log-filter-builder__field-button--disabled': definition.disabled,
                        },
                      ]"
                      :disabled="definition.disabled"
                      type="button"
                      @click="$emit('update:selectedFieldKey', definition.key)"
                    >
                      {{ definition.label }}
                    </button>
                  </div>

                  <div v-if="selectedField" class="log-filter-builder__editor">
                    <div class="log-filter-builder__editor-title">
                      {{ selectedField.label }}
                    </div>

                    <template v-if="selectedField.key === timeFieldKey">
                      <div class="log-filter-builder__time-list">
                        <div v-for="field in timeFields" :key="field.key" class="log-filter-builder__time-item">
                          <span class="log-filter-builder__time-label">{{ field.label }}</span>
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
                      <div class="log-filter-builder__sort-list">
                        <div
                          v-for="(sorter, index) in sorters"
                          :key="`sort-row-${index}`"
                          class="log-filter-builder__sort-row"
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
                          <div class="log-filter-builder__sort-actions">
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

              <t-button theme="default" variant="dashed">{{ addFilterLabel }}</t-button>
            </t-popup>
          </div>
        </section>

        <div v-if="presets.length" class="log-filter-builder__preset-row">
          <span class="log-filter-builder__preset-label">{{ presetLabel }}</span>
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

        <div v-if="tags.length" class="log-filter-builder__tag-row">
          <t-tag
            v-for="tag in tags"
            :key="tag.key"
            :closable="tag.closable !== false"
            max-width="280"
            size="small"
            theme="primary"
            variant="light-outline"
            @close="$emit('close-tag', tag.key)"
          >
            {{ tag.label }}
          </t-tag>
        </div>
      </div>
    </template>
  </management-toolbar>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

import { ManagementToolbar } from '@/shared/components/management';

import type {
  LogFilterFieldDefinition,
  LogFilterPreset,
  LogFilterTag,
  LogSortItem,
  LogSortOption,
  LogTimeRangeField,
} from './log-filter-builder';

const props = defineProps<{
  activePreset: string;
  addFilterLabel: string;
  addSorterLabel: string;
  builderHint: string;
  builderTitle: string;
  fieldValues: Record<string, string | string[]>;
  fields: LogFilterFieldDefinition[];
  filtersGroupLabel: string;
  keyword: string;
  keywordPlaceholder: string;
  loading?: boolean;
  moveDownLabel: string;
  moveUpLabel: string;
  presetLabel: string;
  presets: LogFilterPreset[];
  removeSorterLabel: string;
  resetLabel: string;
  searchLabel: string;
  selectedFieldKey: string;
  sortAddDisabled?: boolean;
  sortDirectionOptions: LogSortOption[];
  sortDirectionPlaceholder: string;
  sortFieldOptionsByIndex: LogSortOption[][];
  sortFieldKey: string;
  sortFieldPlaceholder: string;
  sortMoveDownDisabled?: boolean[];
  sortMoveUpDisabled?: boolean[];
  sorters: LogSortItem[];
  tags: LogFilterTag[];
  timeFieldKey: string;
  timeFields: LogTimeRangeField[];
}>();

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

const selectedField = computed(() => props.fields.find((field) => field.key === props.selectedFieldKey) ?? null);

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
.log-filter-builder {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.log-filter-builder__top-row,
.log-filter-builder__group-body,
.log-filter-builder__preset-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.log-filter-builder__keyword {
  flex: 1 1 340px;
  min-width: 240px;
}

.log-filter-builder__actions {
  display: flex;
  gap: 12px;
  margin-left: auto;
}

.log-filter-builder__group {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  padding: 12px 14px;
}

.log-filter-builder__group-header {
  margin-bottom: 10px;
}

.log-filter-builder__group-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.log-filter-builder__popup {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(180px, 220px) minmax(320px, 420px);
  padding: 8px;
}

.log-filter-builder__header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  grid-column: 1 / -1;
}

.log-filter-builder__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.log-filter-builder__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-filter-builder__field-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-filter-builder__field-button {
  background: var(--td-bg-color-container-hover);
  border: 1px solid transparent;
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  cursor: pointer;
  padding: 10px 12px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.log-filter-builder__field-button--active {
  background: color-mix(in srgb, var(--td-brand-color-light) 40%, var(--td-bg-color-container) 60%);
  border-color: var(--td-brand-color);
}

.log-filter-builder__field-button--disabled {
  color: var(--td-text-color-disabled);
  cursor: not-allowed;
}

.log-filter-builder__editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.log-filter-builder__editor-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.log-filter-builder__time-list,
.log-filter-builder__sort-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-filter-builder__time-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-filter-builder__time-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-filter-builder__sort-row {
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(120px, 1fr) minmax(120px, 1fr);
}

.log-filter-builder__sort-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  grid-column: 1 / -1;
}

.log-filter-builder__preset-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.log-filter-builder__tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

@media (width <= 900px) {
  .log-filter-builder__popup {
    grid-template-columns: 1fr;
  }
}
</style>
