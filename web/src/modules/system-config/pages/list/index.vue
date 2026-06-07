<template>
  <section class="system-config-page" data-page-type="settings">
    <header class="system-config-page__header">
      <div>
        <span class="system-config-page__eyebrow">{{ t('systemConfig.list.eyebrow') }}</span>
        <h1>{{ t('systemConfig.list.title') }}</h1>
        <p>{{ t('systemConfig.list.description') }}</p>
      </div>
      <t-button theme="primary" :loading="loading" @click="refreshConfigs">
        <template #icon><refresh-icon /></template>
        {{ t('systemConfig.list.refresh') }}
      </t-button>
    </header>

    <t-alert
      v-if="errorMessage"
      theme="error"
      :title="t('systemConfig.list.loadError')"
      :message="errorMessage"
      class="system-config-page__alert"
    />

    <t-loading :loading="loading">
      <div class="system-config-layout">
        <aside class="system-config-groups">
          <button
            v-for="group in groupedConfigs"
            :key="group.key"
            type="button"
            :class="['system-config-group', { 'system-config-group--active': group.key === activeGroupKey }]"
            @click="activeGroupKey = group.key"
          >
            <span>{{ group.label }}</span>
            <small>{{ group.technicalKey }}</small>
            <small v-if="group.items.length > 1">
              {{ t('systemConfig.list.groupConfigCount', { count: group.items.length }) }}
            </small>
          </button>
        </aside>

        <main class="system-config-content">
          <div v-if="activeGroup" class="system-config-content__head">
            <div>
              <h2>{{ activeGroup.label }}</h2>
              <p>{{ activeGroup.technicalKey }}</p>
            </div>
            <t-tag v-if="activeGroupOverrideCount > 0" variant="light" theme="primary">
              {{ t('systemConfig.list.overrideCount', { count: activeGroupOverrideCount }) }}
            </t-tag>
          </div>

          <div v-if="activeGroup?.items.length" class="system-config-list">
            <t-card v-for="item in activeGroup.items" :key="item.key" class="system-config-item" bordered>
              <div class="system-config-item__main">
                <div class="system-config-item__title-row">
                  <div>
                    <h3>{{ configTitle(item) }}</h3>
                    <p>{{ configDescription(item) }}</p>
                  </div>
                  <t-space size="small" break-line>
                    <t-tag :theme="configStatus(item).theme" variant="light">
                      {{ configStatus(item).label }}
                    </t-tag>
                    <t-tag v-if="item.sensitive" theme="danger" variant="light">
                      {{ t('systemConfig.list.tags.sensitive') }}
                    </t-tag>
                    <t-tag v-if="item.restart_required" theme="primary" variant="light">
                      {{ t('systemConfig.list.tags.restartRequired') }}
                    </t-tag>
                  </t-space>
                </div>

                <div class="system-config-summary">
                  <section class="system-config-summary__cell">
                    <span>{{ t('systemConfig.list.status.title') }}</span>
                    <div>
                      <t-tag :theme="configStatus(item).theme" variant="light">
                        {{ configStatus(item).label }}
                      </t-tag>
                      <strong>{{ configStatus(item).description }}</strong>
                    </div>
                  </section>
                  <section class="system-config-summary__cell">
                    <span>{{ t('systemConfig.list.source.title') }}</span>
                    <strong>{{ configSourceLabel(item) }}</strong>
                  </section>
                </div>

                <div class="system-config-values">
                  <section
                    v-for="valueSection in valueSections(item)"
                    :key="valueSection.field"
                    class="system-config-value"
                  >
                    <header>
                      <h4>{{ valueSection.title }}</h4>
                    </header>
                    <dl class="system-config-value__rows">
                      <template v-for="row in valueSection.rows" :key="row.key">
                        <dt>{{ row.label }}</dt>
                        <dd>
                          <strong>{{ row.value }}</strong>
                          <small v-if="row.description">{{ row.description }}</small>
                        </dd>
                      </template>
                    </dl>
                    <t-collapse
                      v-if="valueSection.json"
                      borderless
                      expand-icon-placement="right"
                      class="system-config-json"
                    >
                      <t-collapse-panel :value="valueSection.field" :header="t('systemConfig.list.viewJson')">
                        <pre>{{ valueSection.json }}</pre>
                      </t-collapse-panel>
                    </t-collapse>
                  </section>
                </div>

                <section class="system-config-technical">
                  <span>{{ t('systemConfig.list.technicalId') }}</span>
                  <code>{{ item.key }}</code>
                </section>

                <div class="system-config-item__actions">
                  <t-button
                    v-permission="permissionCodes.WRITE"
                    theme="primary"
                    variant="outline"
                    @click="openEditor(item)"
                  >
                    <template #icon><edit-icon /></template>
                    {{ t('systemConfig.list.edit') }}
                  </t-button>
                  <t-popconfirm
                    v-if="item.has_override"
                    theme="warning"
                    :content="t('systemConfig.list.resetConfirm')"
                    :confirm-btn="t('systemConfig.list.reset')"
                    :cancel-btn="t('systemConfig.list.cancel')"
                    @confirm="resetConfigOverride(item)"
                  >
                    <t-button
                      v-permission="permissionCodes.WRITE"
                      theme="default"
                      variant="outline"
                      :loading="resettingKey === item.key"
                    >
                      <template #icon><rollback-icon /></template>
                      {{ t('systemConfig.list.reset') }}
                    </t-button>
                  </t-popconfirm>
                </div>
              </div>
            </t-card>
          </div>

          <t-empty
            v-else
            :title="t('systemConfig.list.emptyTitle')"
            :description="t('systemConfig.list.emptyDescription')"
          >
            <template #action>
              <t-button theme="primary" variant="outline" @click="refreshConfigs">
                {{ t('systemConfig.list.refresh') }}
              </t-button>
            </template>
          </t-empty>
        </main>
      </div>
    </t-loading>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editorTitle"
      :confirm-btn="t('systemConfig.list.save')"
      :cancel-btn="t('systemConfig.list.cancel')"
      :confirm-loading="saving"
      width="680px"
      destroy-on-close
      @confirm="saveEditor"
    >
      <div v-if="editingItem" class="system-config-editor">
        <t-alert v-if="editingItem.sensitive" theme="warning" :message="t('systemConfig.list.sensitiveEditHint')" />
        <t-form :data="editorForm" label-align="top">
          <json-schema-value-fields
            v-model="editorForm.value"
            :root-schema="editingSchema"
            :labels="schemaLabels"
            :title-resolver="schemaFieldTitle"
            :description-resolver="schemaFieldDescription"
            :placeholder-resolver="schemaFieldPlaceholder"
            :unit-resolver="schemaFieldUnit"
            :option-label-resolver="schemaOptionLabel"
          />
        </t-form>
        <section class="system-config-editor__preview">
          <strong>{{ t('systemConfig.list.previewTitle') }}</strong>
          <pre>{{ editorPreview }}</pre>
        </section>
      </div>
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import { EditIcon, RefreshIcon, RollbackIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  type ConfigSchemaField,
  getConfigSchemaFields,
  JsonSchemaValueFields,
  parseConfigSchema,
} from '@/shared/schema-form';
import { formatJsonValue, isJsonRecord, parseJsonValue, valuePreview } from '@/shared/schema-form/json';
import type { ApiRequestError } from '@/types/axios';

import { getSystemConfigs, resetSystemConfig, updateSystemConfig } from '../../api/system-config';
import { SYSTEM_CONFIG_PERMISSION_CODE } from '../../contract/permissions';
import type { SystemConfigItem } from '../../types/system-config';

defineOptions({
  name: 'SystemConfigListPage',
});

type ConfigGroup = {
  key: string;
  label: string;
  technicalKey: string;
  items: SystemConfigItem[];
};

type ConfigValueField = 'effective_value' | 'default_value';

type ConfigValueRow = {
  key: string;
  label: string;
  description: string;
  value: string;
};

type ConfigValueSection = {
  field: ConfigValueField;
  title: string;
  rows: ConfigValueRow[];
  json: string;
};

const { t, te } = useI18n();
const permissionCodes = SYSTEM_CONFIG_PERMISSION_CODE;
const items = ref<SystemConfigItem[]>([]);
const loading = ref(false);
const saving = ref(false);
const resettingKey = ref('');
const errorMessage = ref('');
const activeGroupKey = ref('');
const editorVisible = ref(false);
const editingItem = ref<SystemConfigItem | null>(null);
const editorForm = reactive<{ value: unknown }>({ value: undefined });

const schemaLabels = computed(() => ({
  invalidJson: t('systemConfig.list.schema.invalidJson'),
  jsonPlaceholder: t('systemConfig.list.schema.jsonPlaceholder'),
  numberPlaceholder: t('systemConfig.list.schema.numberPlaceholder'),
  selectPlaceholder: t('systemConfig.list.schema.selectPlaceholder'),
  stringPlaceholder: t('systemConfig.list.schema.stringPlaceholder'),
  value: t('systemConfig.list.schema.value'),
}));

const groupedConfigs = computed<ConfigGroup[]>(() => {
  const groupMap = new Map<string, ConfigGroup>();
  const sortedItems = [...items.value].sort((left, right) => {
    const orderDelta = (left.order ?? 0) - (right.order ?? 0);
    return orderDelta || configTitle(left).localeCompare(configTitle(right));
  });

  for (const item of sortedItems) {
    const key = `${item.module}:${item.group || 'default'}`;
    const label = groupLabel(item);
    const group = groupMap.get(key) ?? { key, label, technicalKey: technicalGroupKey(item), items: [] };
    group.items.push(item);
    groupMap.set(key, group);
  }

  return [...groupMap.values()];
});

const activeGroup = computed(() => groupedConfigs.value.find((group) => group.key === activeGroupKey.value) ?? null);
const activeGroupOverrideCount = computed(
  () => activeGroup.value?.items.filter((item) => item.has_override).length ?? 0,
);
const editingSchema = computed(() => parseConfigSchema(editingItem.value?.config_schema));
const editorTitle = computed(() =>
  editingItem.value ? t('systemConfig.list.editorTitle', { title: configTitle(editingItem.value) }) : '',
);
const editorPreview = computed(() => formatJsonValue(editorForm.value) || t('systemConfig.list.emptyValue'));

onMounted(refreshConfigs);

async function refreshConfigs() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getSystemConfigs();
    items.value = response.items ?? [];
    if (!activeGroupKey.value || !groupedConfigs.value.some((group) => group.key === activeGroupKey.value)) {
      activeGroupKey.value = groupedConfigs.value[0]?.key ?? '';
    }
  } catch (error) {
    errorMessage.value = readableError(error, t('systemConfig.list.loadError'));
  } finally {
    loading.value = false;
  }
}

function openEditor(item: SystemConfigItem) {
  editingItem.value = item;
  editorForm.value = initialEditorValue(item);
  editorVisible.value = true;
}

async function saveEditor() {
  if (!editingItem.value) {
    return;
  }

  saving.value = true;
  try {
    const updated = await updateSystemConfig(editingItem.value.key, { value: editorForm.value });
    upsertConfig(updated);
    editingItem.value = updated;
    editorVisible.value = false;
    MessagePlugin.success(t('systemConfig.list.saveSuccess'));
  } catch (error) {
    MessagePlugin.error(readableError(error, t('systemConfig.list.saveError')));
  } finally {
    saving.value = false;
  }
}

async function resetConfigOverride(item: SystemConfigItem) {
  resettingKey.value = item.key;
  try {
    const updated = await resetSystemConfig(item.key);
    upsertConfig(updated);
    MessagePlugin.success(t('systemConfig.list.resetSuccess'));
  } catch (error) {
    MessagePlugin.error(readableError(error, t('systemConfig.list.resetError')));
  } finally {
    resettingKey.value = '';
  }
}

function upsertConfig(item: SystemConfigItem) {
  const nextItems = [...items.value];
  const index = nextItems.findIndex((candidate) => candidate.key === item.key);
  if (index >= 0) {
    nextItems[index] = item;
  } else {
    nextItems.push(item);
  }
  items.value = nextItems;
}

function initialEditorValue(item: SystemConfigItem) {
  if (item.sensitive) {
    return item.type === 'boolean' ? false : '';
  }

  return (
    parseJsonValue(item.override_value) ?? parseJsonValue(item.effective_value) ?? parseJsonValue(item.default_value)
  );
}

function configTitle(item: SystemConfigItem) {
  return resolveI18nText(item.title_key, item.title, item.key);
}

function configDescription(item: SystemConfigItem) {
  return resolveI18nText(item.description_key, item.description, t('systemConfig.list.noDescription'));
}

function groupLabel(item: SystemConfigItem) {
  return resolveI18nText(item.group_key, item.group_label, technicalGroupKey(item));
}

function technicalGroupKey(item: SystemConfigItem) {
  return t('systemConfig.list.groupLabel', { module: item.module, group: item.group || 'default' });
}

function configStatus(item: SystemConfigItem) {
  if (item.has_override) {
    return {
      label: t('systemConfig.list.status.overridden'),
      description: t('systemConfig.list.status.overriddenDescription'),
      theme: 'primary' as const,
    };
  }

  return {
    label: t('systemConfig.list.status.default'),
    description: t('systemConfig.list.status.defaultDescription'),
    theme: 'default' as const,
  };
}

function configSourceLabel(item: SystemConfigItem) {
  return item.has_override
    ? t('systemConfig.list.source.values.administrator_override')
    : t('systemConfig.list.source.values.default');
}

function valueSections(item: SystemConfigItem): ConfigValueSection[] {
  return [
    buildValueSection(item, 'effective_value', t('systemConfig.list.values.effective')),
    buildValueSection(item, 'default_value', t('systemConfig.list.values.default')),
  ];
}

function buildValueSection(item: SystemConfigItem, field: ConfigValueField, title: string): ConfigValueSection {
  if (item.masked) {
    const maskedValue = item.masked_placeholder || t('systemConfig.list.masked');
    return {
      field,
      title,
      rows: [{ key: field, label: title, description: '', value: maskedValue }],
      json: '',
    };
  }

  const parsed = parseJsonValue(item[field]);
  const schema = parseConfigSchema(item.config_schema);
  const fields = getConfigSchemaFields(schema);
  const rows = isJsonRecord(parsed) && fields.length > 0 ? structuredValueRows(parsed, fields) : [];
  const fallbackRows =
    rows.length > 0
      ? rows
      : [
          {
            key: field,
            label: schema.title ? resolveI18nText(schema.xI18n?.titleKey, schema.title, title) : title,
            description: resolveI18nText(schema.xI18n?.descriptionKey, schema.description, ''),
            value: valuePreview(parsed, t('systemConfig.list.emptyValue'), booleanLabel),
          },
        ];

  return {
    field,
    title,
    rows: fallbackRows,
    json: formatJsonValue(parsed),
  };
}

function structuredValueRows(value: Record<string, unknown>, fields: ConfigSchemaField[]): ConfigValueRow[] {
  return fields.map((field) => {
    const unit = schemaFieldUnit(field);
    const displayValue = valuePreview(value[field.key], t('systemConfig.list.emptyValue'), booleanLabel);
    return {
      key: field.key,
      label: schemaFieldTitle(field),
      description: schemaFieldDescription(field),
      value: unit ? `${displayValue} ${unit}` : displayValue,
    };
  });
}

function booleanLabel(value: boolean) {
  return value ? t('systemConfig.list.boolean.true') : t('systemConfig.list.boolean.false');
}

function schemaFieldTitle(field: ConfigSchemaField) {
  return resolveI18nText(field.schema.xI18n?.titleKey, field.schema.title, field.key);
}

function schemaFieldDescription(field: ConfigSchemaField) {
  return resolveI18nText(field.schema.xI18n?.descriptionKey, field.schema.description, '');
}

function schemaFieldPlaceholder(field: ConfigSchemaField) {
  return resolveI18nText(field.schema.xI18n?.placeholderKey, field.schema.placeholder, '');
}

function schemaFieldUnit(field: ConfigSchemaField) {
  return resolveI18nText(field.schema.xI18n?.unitKey, undefined, '');
}

function schemaOptionLabel(field: ConfigSchemaField, option: string | number | boolean) {
  const optionText = field.schema.enumLabels?.[String(option)];
  return resolveI18nText(optionText?.labelKey, optionText?.label, String(option));
}

function resolveI18nText(key?: string, fallback?: string, rawFallback = '') {
  return key && te(key) ? t(key) : fallback || rawFallback;
}

function readableError(error: unknown, fallback: string) {
  const requestError = error as Partial<ApiRequestError>;
  return typeof requestError.message === 'string' && requestError.message.trim() ? requestError.message : fallback;
}
</script>
<style scoped>
.system-config-page {
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.system-config-page__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.system-config-page__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
}

.system-config-page__header h1,
.system-config-content__head h2,
.system-config-item h3 {
  margin: 0;
}

.system-config-page__header h1 {
  font: var(--td-font-headline-medium);
}

.system-config-page__header p,
.system-config-content__head p,
.system-config-item p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
}

.system-config-layout {
  align-items: flex-start;
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
}

.system-config-groups {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-12);
  position: sticky;
  top: var(--graft-density-gap-16);
}

.system-config-group {
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-height: 56px;
  padding: var(--graft-density-gap-8);
  text-align: left;
}

.system-config-group:hover,
.system-config-group--active {
  background: var(--td-brand-color-light);
  border-color: var(--td-brand-color);
}

.system-config-group small {
  color: var(--td-text-color-secondary);
}

.system-config-content {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.system-config-content__head,
.system-config-item__title-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.system-config-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.system-config-item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
}

.system-config-item :deep(.t-card__body) {
  padding: var(--graft-density-gap-16);
}

.system-config-item__main,
.system-config-editor {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.system-config-item__actions {
  display: flex;
  flex-flow: row wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

.system-config-summary,
.system-config-values {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.system-config-summary {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.system-config-summary__cell,
.system-config-value,
.system-config-technical {
  background: var(--td-bg-color-page);
  border-radius: var(--td-radius-small);
  padding: var(--graft-density-gap-12);
}

.system-config-summary__cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.system-config-summary__cell > span,
.system-config-value h4,
.system-config-technical span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.system-config-summary__cell > div {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.system-config-values {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.system-config-value {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.system-config-value__rows {
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-16);
  grid-template-columns: minmax(120px, max-content) minmax(0, 1fr);
  margin: 0;
}

.system-config-value__rows dt {
  color: var(--td-text-color-secondary);
}

.system-config-value__rows dd {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  margin: 0;
  min-width: 0;
}

.system-config-value__rows strong,
.system-config-technical code {
  overflow-wrap: anywhere;
}

.system-config-value__rows small {
  color: var(--td-text-color-placeholder);
}

.system-config-json :deep(.t-collapse-panel__content) {
  padding: 0;
}

.system-config-technical {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.system-config-json pre,
.system-config-editor__preview pre {
  border-radius: var(--td-radius-small);
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  margin: var(--graft-density-gap-8) 0 0;
  max-height: 220px;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  white-space: pre-wrap;
}

.system-config-json pre {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
}

.system-config-editor__preview pre {
  background: var(--td-bg-color-page);
}

@media (width <= 900px) {
  .system-config-page__header,
  .system-config-layout,
  .system-config-summary,
  .system-config-values {
    display: flex;
    flex-direction: column;
  }

  .system-config-groups {
    position: static;
    width: 100%;
  }

  .system-config-item__actions {
    flex-flow: row wrap;
  }
}
</style>
