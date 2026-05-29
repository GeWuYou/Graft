<template>
  <management-toolbar>
    <template #filters>
      <div class="audit-filters">
        <div class="audit-filters__top-row">
          <t-input
            :model-value="modelValue.keyword"
            class="audit-filters__keyword"
            clearable
            :placeholder="t('audit.logList.filters.keywordPlaceholder')"
            @update:model-value="updateField('keyword', $event)"
          />
          <t-date-range-picker
            :model-value="modelValue.createdRange"
            allow-input
            clearable
            class="audit-filters__date"
            enable-time-picker
            format="YYYY-MM-DD HH:mm:ss"
            :placeholder="dateRangePlaceholder"
            @update:model-value="updateField('createdRange', $event)"
          />
          <div class="audit-filters__actions">
            <t-button theme="primary" :loading="loading" @click="$emit('search')">
              {{ t('audit.logList.actions.search') }}
            </t-button>
            <t-button theme="default" variant="outline" @click="$emit('reset')">
              {{ t('audit.logList.actions.reset') }}
            </t-button>
          </div>
        </div>

        <div v-if="activeFilterTags.length" class="audit-filters__tag-row">
          <t-tag
            v-for="tag in activeFilterTags"
            :key="tag.key"
            closable
            max-width="240"
            theme="primary"
            variant="light-outline"
            @close="clearField(tag.key)"
          >
            {{ tag.label }}
          </t-tag>
        </div>

        <div class="audit-filters__bottom-row">
          <t-popup
            v-model:visible="builderVisible"
            attach="body"
            destroy-on-close
            overlay-class-name="audit-filter-builder-popup"
            placement="bottom-left"
            trigger="click"
          >
            <template #content>
              <div class="audit-filter-builder">
                <div class="audit-filter-builder__header">
                  <span class="audit-filter-builder__title">{{ t('audit.logList.builder.title') }}</span>
                  <span class="audit-filter-builder__hint">{{ t('audit.logList.builder.hint') }}</span>
                </div>

                <div class="audit-filter-builder__field-list">
                  <button
                    v-for="definition in availableDefinitions"
                    :key="definition.key"
                    class="audit-filter-builder__field-button"
                    type="button"
                    @click="selectDefinition(definition.key)"
                  >
                    {{ definition.fieldLabel }}
                  </button>
                </div>

                <div v-if="selectedDefinition" class="audit-filter-builder__editor">
                  <div class="audit-filter-builder__editor-title">
                    {{ selectedDefinition.fieldLabel }}
                  </div>
                  <t-select
                    v-if="selectedDefinition.kind === 'select'"
                    :model-value="selectValue(selectedDefinition.key)"
                    clearable
                    :options="selectedDefinition.options"
                    :placeholder="selectedDefinition.placeholder"
                    @update:model-value="updateField(selectedDefinition.key, normalizeSelectValue($event))"
                  />
                  <t-input
                    v-else
                    :model-value="textValue(selectedDefinition.key)"
                    clearable
                    :placeholder="selectedDefinition.placeholder"
                    @update:model-value="updateField(selectedDefinition.key, normalizeTextValue($event))"
                  />
                </div>
              </div>
            </template>

            <t-button theme="default" variant="dashed"> + {{ t('audit.logList.actions.addFilter') }} </t-button>
          </t-popup>

          <div class="audit-filters__preset-row">
            <span class="audit-filters__preset-label">{{ t('audit.logList.presets.label') }}</span>
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
        </div>
      </div>
    </template>
  </management-toolbar>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementToolbar } from '@/shared/components/management';

import type { AuditClientFilterState } from '../shared/presentation';

type PresetKey = 'all' | 'today-anomalies' | 'permission-denied' | 'sensitive-ops' | 'auth-failed' | 'high-risk';

type FilterKey = Exclude<keyof AuditClientFilterState, 'keyword' | 'createdRange'>;
type SelectFilterKey = 'action' | 'source' | 'resourceType' | 'result' | 'riskLevel';
type TextFilterKey = Exclude<FilterKey, SelectFilterKey>;

type FilterOption = {
  label: string;
  value: string;
};

type BaseDefinition<Key extends FilterKey> = {
  key: Key;
  fieldLabel: string;
  placeholder: string;
};

type SelectDefinition = BaseDefinition<SelectFilterKey> & {
  kind: 'select';
  options: FilterOption[];
};

type TextDefinition = BaseDefinition<TextFilterKey> & {
  kind: 'text';
};

type FilterDefinition = SelectDefinition | TextDefinition;

const props = defineProps<{
  activePreset: PresetKey;
  loading?: boolean;
  modelValue: AuditClientFilterState;
  presets: { key: PresetKey; title: string }[];
}>();

const emit = defineEmits<{
  (e: 'apply-preset', preset: PresetKey): void;
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:modelValue', value: AuditClientFilterState): void;
}>();

const { t } = useI18n();

const builderVisible = ref(false);
const selectedDefinitionKey = ref<FilterKey>('actor');

const actionOptions = computed<FilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.auth'), value: 'auth' },
  { label: t('audit.logList.filterOptions.role'), value: 'role' },
  { label: t('audit.logList.filterOptions.permission'), value: 'permission' },
  { label: t('audit.logList.filterOptions.session'), value: 'session' },
]);

const sourceOptions = computed<FilterOption[]>(() => [
  { label: t('audit.common.source.REQUEST'), value: 'REQUEST' },
  { label: t('audit.common.source.SECURITY_EVENT'), value: 'SECURITY_EVENT' },
  { label: t('audit.common.source.DOMAIN_EVENT'), value: 'DOMAIN_EVENT' },
]);

const resourceTypeOptions = computed<FilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.userResource'), value: 'user' },
  { label: t('audit.logList.filterOptions.roleResource'), value: 'role' },
  { label: t('audit.logList.filterOptions.permissionResource'), value: 'permission' },
  { label: t('audit.logList.filterOptions.authResource'), value: 'auth' },
]);

const resultOptions = computed<FilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'SUCCESS' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'FAILED' },
  { label: t('audit.logList.filterOptions.DENIED'), value: 'DENIED' },
  { label: t('audit.logList.filterOptions.ERROR'), value: 'ERROR' },
]);

const riskOptions = computed<FilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.LOW'), value: 'LOW' },
  { label: t('audit.logList.filterOptions.MEDIUM'), value: 'MEDIUM' },
  { label: t('audit.logList.filterOptions.HIGH'), value: 'HIGH' },
  { label: t('audit.logList.filterOptions.CRITICAL'), value: 'CRITICAL' },
]);

const definitions = computed<FilterDefinition[]>(() => [
  {
    key: 'action',
    kind: 'select',
    fieldLabel: t('audit.logList.builder.fields.action'),
    placeholder: t('audit.logList.filters.actionPlaceholder'),
    options: actionOptions.value,
  },
  {
    key: 'result',
    kind: 'select',
    fieldLabel: t('audit.logList.builder.fields.result'),
    placeholder: t('audit.logList.filters.resultPlaceholder'),
    options: resultOptions.value,
  },
  {
    key: 'riskLevel',
    kind: 'select',
    fieldLabel: t('audit.logList.builder.fields.riskLevel'),
    placeholder: t('audit.logList.filters.riskPlaceholder'),
    options: riskOptions.value,
  },
  {
    key: 'source',
    kind: 'select',
    fieldLabel: t('audit.logList.builder.fields.source'),
    placeholder: t('audit.logList.filters.sourcePlaceholder'),
    options: sourceOptions.value,
  },
  {
    key: 'actor',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.actor'),
    placeholder: t('audit.logList.filters.actorPlaceholder'),
  },
  {
    key: 'resourceName',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.resourceName'),
    placeholder: t('audit.logList.filters.resourceNamePlaceholder'),
  },
  {
    key: 'resourceType',
    kind: 'select',
    fieldLabel: t('audit.logList.builder.fields.resourceType'),
    placeholder: t('audit.logList.filters.resourceTypePlaceholder'),
    options: resourceTypeOptions.value,
  },
  {
    key: 'requestId',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.requestId'),
    placeholder: t('audit.logList.filters.requestIdPlaceholder'),
  },
  {
    key: 'traceId',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.traceId'),
    placeholder: t('audit.logList.filters.traceIdPlaceholder'),
  },
  {
    key: 'session',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.session'),
    placeholder: t('audit.logList.filters.sessionPlaceholder'),
  },
  {
    key: 'resourceId',
    kind: 'text',
    fieldLabel: t('audit.logList.builder.fields.resourceId'),
    placeholder: t('audit.logList.filters.resourceIdPlaceholder'),
  },
]);

const definitionMap = computed(() => new Map(definitions.value.map((item) => [item.key, item])));
const selectedDefinition = computed(() => definitionMap.value.get(selectedDefinitionKey.value));

const availableDefinitions = computed(() =>
  definitions.value.filter(
    (definition) => !isFieldActive(definition.key) || definition.key === selectedDefinitionKey.value,
  ),
);

const activeFilterTags = computed(() =>
  definitions.value
    .map((definition) => {
      const label = buildTagLabel(definition);
      return label ? { key: definition.key, label } : null;
    })
    .filter((item): item is { key: FilterKey; label: string } => Boolean(item)),
);

const dateRangePlaceholder = computed(() => [
  t('audit.logList.filters.datePlaceholder'),
  t('audit.logList.filters.datePlaceholder'),
]);

function updateField<Key extends keyof AuditClientFilterState>(key: Key, value: AuditClientFilterState[Key]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function selectDefinition(key: FilterKey) {
  selectedDefinitionKey.value = key;
}

function isFieldActive(key: FilterKey) {
  const value = props.modelValue[key];
  if (key === 'result' || key === 'riskLevel') {
    return value !== 'all';
  }
  return typeof value === 'string' ? Boolean(value.trim()) : Boolean(value);
}

function buildTagLabel(definition: FilterDefinition) {
  const value = props.modelValue[definition.key];
  if (definition.key === 'result' || definition.key === 'riskLevel') {
    if (value === 'all') {
      return '';
    }
  } else if (typeof value === 'string' && !value.trim()) {
    return '';
  }

  const display = definition.kind === 'select' ? optionLabel(definition.options, String(value)) : String(value);
  return `${definition.fieldLabel}：${display}`;
}

function optionLabel(options: FilterOption[], value: string) {
  return options.find((option) => option.value === value)?.label || value;
}

function clearField(key: FilterKey) {
  if (key === 'result') {
    updateField(key, 'all' as AuditClientFilterState[typeof key]);
    return;
  }
  if (key === 'riskLevel') {
    updateField(key, 'all' as AuditClientFilterState[typeof key]);
    return;
  }
  updateField(key, '' as AuditClientFilterState[typeof key]);
}

function textValue(key: TextFilterKey) {
  return props.modelValue[key];
}

function selectValue(key: SelectFilterKey) {
  const value = props.modelValue[key];
  return value === 'all' ? '' : value;
}

function normalizeTextValue(value: string | number | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeSelectValue(value: string | number | Array<string | number> | undefined) {
  return typeof value === 'string' ? value : '';
}
</script>
<style scoped lang="less">
.audit-filters {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.audit-filters__top-row,
.audit-filters__bottom-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.audit-filters__keyword {
  flex: 1 1 340px;
  min-width: 240px;
}

.audit-filters__date {
  flex: 0 1 360px;
  min-width: 240px;
}

.audit-filters__actions {
  display: flex;
  gap: 12px;
  margin-left: auto;
}

.audit-filters__tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.audit-filters__preset-row {
  align-items: center;
  display: flex;
  flex: 1 1 auto;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.audit-filters__preset-label {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.audit-filter-builder {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(160px, 200px) minmax(220px, 320px);
  padding: 8px;
}

.audit-filter-builder__header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  grid-column: 1 / -1;
}

.audit-filter-builder__title {
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.audit-filter-builder__hint {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.audit-filter-builder__field-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-filter-builder__field-button {
  background: var(--td-bg-color-container-hover);
  border: 1px solid transparent;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  padding: 8px 12px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.audit-filter-builder__field-button:hover {
  border-color: var(--td-brand-color);
}

.audit-filter-builder__editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.audit-filter-builder__editor-title {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

@media (width <= 960px) {
  .audit-filters__actions {
    margin-left: 0;
  }

  .audit-filter-builder {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-filters__top-row,
  .audit-filters__bottom-row {
    align-items: stretch;
    flex-direction: column;
  }

  .audit-filters__keyword,
  .audit-filters__date,
  .audit-filters__actions,
  .audit-filters__preset-row {
    min-width: 0;
    width: 100%;
  }

  .audit-filters__actions {
    justify-content: flex-start;
  }
}
</style>
