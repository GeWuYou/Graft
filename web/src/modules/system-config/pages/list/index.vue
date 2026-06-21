<template>
  <section class="system-config-page" data-page-type="settings">
    <page-header
      :source="{
        labelKey: 'menu.server.title',
        fallback: t('systemConfig.list.eyebrow'),
        color: 'var(--td-brand-color-6)',
      }"
      title-key="systemConfig.list.title"
      :title-fallback="t('systemConfig.list.title')"
      description-key="systemConfig.list.description"
      :description-fallback="t('systemConfig.list.description')"
    >
      <template #actions>
        <t-button theme="primary" :loading="loading" @click="refreshConfigs">
          <template #icon><refresh-icon /></template>
          {{ t('systemConfig.list.refresh') }}
        </t-button>
      </template>
    </page-header>

    <t-alert
      v-if="errorMessage"
      theme="error"
      :title="t('systemConfig.list.loadError')"
      :message="errorMessage"
      class="system-config-page__alert"
    />

    <t-loading :loading="loading" class="system-config-workspace">
      <div class="system-config-layout">
        <aside class="system-config-groups system-config-scrollbar">
          <t-input
            v-model="groupSearchKeyword"
            class="system-config-groups__search"
            clearable
            :placeholder="t('systemConfig.list.searchPlaceholder')"
            type="search"
          >
            <template #suffixIcon><search-icon /></template>
          </t-input>
          <t-tree
            :data="domainTree"
            :actived="activeTreeValue"
            :expanded="expandedDomainKeys"
            :empty="t('systemConfig.list.searchEmpty')"
            activable
            hover
            expand-on-click-node
            @active="handleTreeActive"
          >
            <template #label="{ node }">
              <span class="system-config-tree-node">
                <span>{{ node.data.label }}</span>
                <small v-if="node.data.count">
                  {{ t('systemConfig.list.groupConfigCount', { count: node.data.count }) }}
                </small>
              </span>
            </template>
          </t-tree>
        </aside>

        <main class="system-config-content system-config-scrollbar">
          <div v-if="activeGroup" class="system-config-content__head">
            <div>
              <h2>{{ activeGroup.label }}</h2>
              <p>{{ activeGroup.description }}</p>
            </div>
            <t-space size="small" break-line>
              <t-tag variant="light">
                {{ t('systemConfig.list.groupConfigCount', { count: activeGroup.items.length }) }}
              </t-tag>
              <t-tag :theme="activeGroupOverrideCount > 0 ? 'primary' : 'default'" variant="light">
                {{ t('systemConfig.list.overrideCount', { count: activeGroupOverrideCount }) }}
              </t-tag>
            </t-space>
          </div>

          <div v-if="activeConfigCards.length" class="system-config-list">
            <t-card v-for="card in activeConfigCards" :key="card.key" class="system-config-item" bordered>
              <div class="system-config-item__main">
                <div class="system-config-item__title-row">
                  <div>
                    <h3>{{ card.title }}</h3>
                    <p>{{ card.description }}</p>
                  </div>
                  <t-space size="small" break-line>
                    <t-tag :theme="card.effectiveSource.theme" variant="light">
                      {{ card.effectiveSource.label }}
                    </t-tag>
                    <t-tag v-if="card.sensitive" theme="danger" variant="light">
                      {{ t('systemConfig.list.tags.sensitive') }}
                    </t-tag>
                    <t-tag v-if="card.runtimeHot" theme="success" variant="light">
                      {{ t('systemConfig.list.tags.runtimeHot') }}
                    </t-tag>
                    <t-tag v-if="card.restartRequired" theme="primary" variant="light">
                      {{ t('systemConfig.list.tags.restartRequired') }}
                    </t-tag>
                  </t-space>
                </div>

                <div class="system-config-values">
                  <section
                    v-for="valueSection in card.valueSections"
                    :key="valueSection.key"
                    class="system-config-value"
                  >
                    <header>
                      <h4>{{ valueSection.title }}</h4>
                    </header>
                    <dl class="system-config-value__rows">
                      <template v-for="row in valueSection.rows" :key="row.key">
                        <dt>{{ row.label }}</dt>
                        <dd>
                          <config-value-renderer
                            :value="row.rawValue"
                            :schema="row.schema"
                            :unit="row.unit"
                            :empty-value-label="t('systemConfig.list.emptyValue')"
                            :boolean-label-resolver="booleanStateLabel"
                            :schema-description-resolver="schemaDescription"
                            :option-label-resolver="schemaOptionDisplayLabel"
                            :option-description-resolver="schemaOptionDescription"
                          >
                            <template #description="{ description, mode }">
                              <t-tooltip
                                v-if="description && mode === 'tooltip'"
                                :content="description"
                                placement="top"
                                show-arrow
                              >
                                <button
                                  class="system-config-value__info"
                                  type="button"
                                  :aria-label="t('systemConfig.list.valueDescription')"
                                >
                                  <info-circle-icon />
                                </button>
                              </t-tooltip>
                            </template>
                          </config-value-renderer>
                        </dd>
                      </template>
                    </dl>
                    <t-collapse
                      v-if="valueSection.extraRows.length"
                      borderless
                      expand-icon-placement="right"
                      class="system-config-more-fields"
                    >
                      <t-collapse-panel
                        :value="`${valueSection.key}-more`"
                        :header="t('systemConfig.list.values.moreFields', { count: valueSection.extraRows.length })"
                      >
                        <dl class="system-config-value__rows">
                          <template v-for="row in valueSection.extraRows" :key="row.key">
                            <dt>{{ row.label }}</dt>
                            <dd>
                              <config-value-renderer
                                :value="row.rawValue"
                                :schema="row.schema"
                                :unit="row.unit"
                                :empty-value-label="t('systemConfig.list.emptyValue')"
                                :boolean-label-resolver="booleanStateLabel"
                                :schema-description-resolver="schemaDescription"
                                :option-label-resolver="schemaOptionDisplayLabel"
                                :option-description-resolver="schemaOptionDescription"
                              >
                                <template #description="{ description, mode }">
                                  <t-tooltip
                                    v-if="description && mode === 'tooltip'"
                                    :content="description"
                                    placement="top"
                                    show-arrow
                                  >
                                    <button
                                      class="system-config-value__info"
                                      type="button"
                                      :aria-label="t('systemConfig.list.valueDescription')"
                                    >
                                      <info-circle-icon />
                                    </button>
                                  </t-tooltip>
                                </template>
                              </config-value-renderer>
                            </dd>
                          </template>
                        </dl>
                      </t-collapse-panel>
                    </t-collapse>
                  </section>
                </div>

                <div class="system-config-summary">
                  <section class="system-config-summary__cell">
                    <span>{{ t('systemConfig.list.lastModified.title') }}</span>
                    <strong>{{ card.auditLabel }}</strong>
                  </section>
                </div>

                <t-collapse borderless expand-icon-placement="right" class="system-config-advanced">
                  <t-collapse-panel :value="`${card.key}-advanced`" :header="t('systemConfig.list.advanced.title')">
                    <div class="system-config-advanced__grid">
                      <section class="system-config-advanced__section">
                        <div class="system-config-advanced__section-head">
                          <span>{{ t('systemConfig.list.technicalId') }}</span>
                          <t-button theme="default" variant="text" size="small" @click="copyConfigKey(card.key)">
                            <template #icon><copy-icon /></template>
                            {{ t('systemConfig.list.advanced.copyKey') }}
                          </t-button>
                        </div>
                        <code>{{ card.key }}</code>
                      </section>
                      <section v-if="card.advanced.schemaSummary.length" class="system-config-advanced__section">
                        <span>{{ t('systemConfig.list.advanced.schemaSummary') }}</span>
                        <ul>
                          <li v-for="summary in card.advanced.schemaSummary" :key="summary">{{ summary }}</li>
                        </ul>
                      </section>
                      <section v-if="card.advanced.currentJson" class="system-config-advanced__section">
                        <span>{{ t('systemConfig.list.advanced.currentJson') }}</span>
                        <pre>{{ card.advanced.currentJson }}</pre>
                      </section>
                      <section v-if="card.advanced.defaultJson" class="system-config-advanced__section">
                        <span>{{ t('systemConfig.list.advanced.defaultJson') }}</span>
                        <pre>{{ card.advanced.defaultJson }}</pre>
                      </section>
                    </div>
                  </t-collapse-panel>
                </t-collapse>

                <div class="system-config-item__actions">
                  <t-button
                    v-permission="permissionCodes.WRITE"
                    theme="primary"
                    variant="outline"
                    @click="openEditor(card.item)"
                  >
                    <template #icon><edit-icon /></template>
                    {{ t('systemConfig.list.edit') }}
                  </t-button>
                  <t-popconfirm
                    v-if="card.canReset"
                    theme="warning"
                    :content="t('systemConfig.list.resetConfirm')"
                    :confirm-btn="t('systemConfig.list.reset')"
                    :cancel-btn="t('systemConfig.list.cancel')"
                    @confirm="resetConfigOverride(card.item)"
                  >
                    <t-button
                      v-permission="permissionCodes.WRITE"
                      theme="default"
                      variant="outline"
                      :loading="resettingKey === card.key"
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
      v-model:visible="editorDialogVisible"
      :header="editorTitle"
      :confirm-btn="t('systemConfig.list.save')"
      :cancel-btn="t('systemConfig.list.cancel')"
      :confirm-loading="saving"
      width="680px"
      destroy-on-close
      data-testid="config-editor-dialog"
      @confirm="saveEditor"
    >
      <div v-if="editingItem" class="system-config-editor">
        <t-alert v-if="editingItem.sensitive" theme="warning" :message="t('systemConfig.list.sensitiveEditHint')" />
        <t-form :data="editorForm" label-align="top">
          <config-editor-renderer
            v-model="editorForm.value"
            :root-schema="editingSchema"
            :fallback-type="editingItem.type"
            :labels="schemaLabels"
            :title-resolver="schemaFieldTitle"
            :description-resolver="schemaFieldDescription"
            :placeholder-resolver="schemaFieldPlaceholder"
            :unit-resolver="schemaFieldUnit"
            :option-label-resolver="schemaOptionLabel"
          />
        </t-form>
        <t-collapse borderless expand-icon-placement="right" class="system-config-editor__preview">
          <t-collapse-panel value="editor-preview" :header="t('systemConfig.list.previewTitle')">
            <pre>{{ editorPreview }}</pre>
          </t-collapse-panel>
        </t-collapse>
      </div>
    </t-dialog>

    <t-drawer
      v-model:visible="editorDrawerVisible"
      :header="editorTitle"
      size="720px"
      placement="right"
      destroy-on-close
      :footer="false"
      data-testid="config-editor-drawer"
    >
      <div v-if="editingItem" class="system-config-editor system-config-editor--drawer">
        <t-alert v-if="editingItem.sensitive" theme="warning" :message="t('systemConfig.list.sensitiveEditHint')" />
        <t-form :data="editorForm" label-align="top">
          <config-editor-renderer
            v-model="editorForm.value"
            :root-schema="editingSchema"
            :fallback-type="editingItem.type"
            :labels="schemaLabels"
            :title-resolver="schemaFieldTitle"
            :description-resolver="schemaFieldDescription"
            :placeholder-resolver="schemaFieldPlaceholder"
            :unit-resolver="schemaFieldUnit"
            :option-label-resolver="schemaOptionLabel"
          />
        </t-form>
        <t-collapse borderless expand-icon-placement="right" class="system-config-editor__preview">
          <t-collapse-panel value="editor-preview" :header="t('systemConfig.list.previewTitle')">
            <pre>{{ editorPreview }}</pre>
          </t-collapse-panel>
        </t-collapse>
        <div class="system-config-editor__actions">
          <t-button variant="outline" @click="closeEditor">
            {{ t('systemConfig.list.cancel') }}
          </t-button>
          <t-button theme="primary" :loading="saving" data-testid="editor-drawer-save" @click="saveEditor">
            {{ t('systemConfig.list.save') }}
          </t-button>
        </div>
      </div>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { CopyIcon, EditIcon, InfoCircleIcon, RefreshIcon, RollbackIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TreeNodeValue, TreeProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';
import { PageHeader } from '@/shared/components/page';
import { copyText } from '@/shared/observability';
import {
  configEditorContainer,
  ConfigEditorRenderer,
  type ConfigSchema,
  type ConfigSchemaField,
  type ConfigSchemaProperty,
  ConfigValueRenderer,
  getConfigSchemaFields,
  parseConfigSchema,
  validateConfigEditorValue,
} from '@/shared/schema-form';
import { formatJsonValue, isJsonRecord, parseJsonValue } from '@/shared/schema-form/json';
import type { ApiRequestError } from '@/types/axios';

import { getSystemConfigs, resetSystemConfig, updateSystemConfig } from '../../api/system-config';
import { SYSTEM_CONFIG_PERMISSION_CODE } from '../../contract/permissions';
import type { SystemConfigItem } from '../../types/system-config';

defineOptions({
  name: 'SystemConfigListPage',
});

type ConfigGroup = {
  key: string;
  domainKey: string;
  label: string;
  description: string;
  items: SystemConfigItem[];
  searchText: string;
};

type ConfigDomain = {
  key: string;
  label: string;
  groups: ConfigGroup[];
};

type ConfigValueField = 'effective_value' | 'default_value';

type ConfigValueRow = {
  key: string;
  label: string;
  rawValue: unknown;
  schema?: ConfigSchemaProperty;
  unit?: string;
};

type ConfigValueSection = {
  key: ConfigValueField;
  title: string;
  rows: ConfigValueRow[];
  extraRows: ConfigValueRow[];
};

type ConfigCardVM = {
  item: SystemConfigItem;
  key: string;
  title: string;
  description: string;
  effectiveSource: {
    label: string;
    description: string;
    theme: 'default' | 'primary';
  };
  sensitive: boolean;
  runtimeHot: boolean;
  restartRequired: boolean;
  auditLabel: string;
  valueSections: ConfigValueSection[];
  advanced: {
    currentJson: string;
    defaultJson: string;
    schemaSummary: string[];
  };
  canReset: boolean;
};

type EditorContainer = 'dialog' | 'drawer';

const CORE_VALUE_ROW_LIMIT = 3;

const { getLocaleMessage, locale, t, te } = useI18n();
const permissionCodes = SYSTEM_CONFIG_PERMISSION_CODE;
const items = ref<SystemConfigItem[]>([]);
const loading = ref(false);
const saving = ref(false);
const resettingKey = ref('');
const errorMessage = ref('');
const activeGroupKey = ref('');
const expandedDomainKeys = ref<TreeNodeValue[]>([]);
const groupSearchKeyword = ref('');
const editorVisible = ref(false);
const editingItem = ref<SystemConfigItem | null>(null);
const editorForm = reactive<{ value: unknown }>({ value: undefined });

const schemaLabels = computed(() => ({
  invalidJson: t('systemConfig.list.schema.invalidJson'),
  invalidValue: t('systemConfig.list.schema.invalidValue'),
  jsonPlaceholder: t('systemConfig.list.schema.jsonPlaceholder'),
  numberPlaceholder: t('systemConfig.list.schema.numberPlaceholder'),
  selectPlaceholder: t('systemConfig.list.schema.selectPlaceholder'),
  stringPlaceholder: t('systemConfig.list.schema.stringPlaceholder'),
  value: t('systemConfig.list.schema.value'),
}));

const domains = computed<ConfigDomain[]>(() => {
  const domainMap = new Map<string, ConfigDomain>();
  const sortedItems = [...items.value].sort((left, right) => {
    const orderDelta = (left.order ?? 0) - (right.order ?? 0);
    return orderDelta || configTitle(left).localeCompare(configTitle(right));
  });

  for (const item of sortedItems) {
    const domainKey = configDomainKey(item);
    const domain = domainMap.get(domainKey) ?? {
      key: domainKey,
      label: domainLabel(item),
      groups: [],
    };
    const groupKey = `${domainKey}:${item.module}:${item.group || 'default'}`;
    let group = domain.groups.find((candidate) => candidate.key === groupKey);
    if (!group) {
      group = {
        key: groupKey,
        domainKey,
        label: groupLabel(item),
        description: groupDescription(item),
        items: [],
        searchText: '',
      };
      domain.groups.push(group);
    }
    group.items.push(item);
    group.searchText = buildGroupSearchText(group, item, group.searchText);
    domainMap.set(domainKey, domain);
  }

  return [...domainMap.values()];
});

const groupedConfigs = computed(() => domains.value.flatMap((domain) => domain.groups));
const normalizedGroupSearchKeyword = computed(() => normalizeSearchText(groupSearchKeyword.value));
const filteredDomains = computed<ConfigDomain[]>(() => {
  const keyword = normalizedGroupSearchKeyword.value;
  if (!keyword) {
    return domains.value;
  }

  return domains.value
    .map((domain) => ({
      ...domain,
      groups: domain.groups.filter((group) => group.searchText.includes(keyword)),
    }))
    .filter((domain) => domain.groups.length > 0);
});
const domainTree = computed<TreeProps['data']>(() =>
  filteredDomains.value.map((domain) => ({
    value: domain.key,
    label: domain.label,
    children: domain.groups.map((group) => ({
      value: group.key,
      label: group.label,
      count: group.items.length,
    })),
  })),
);
const activeTreeValue = computed(() => (activeGroupKey.value ? [activeGroupKey.value] : []));
const activeGroup = computed(() => groupedConfigs.value.find((group) => group.key === activeGroupKey.value) ?? null);
const activeConfigCards = computed(() => activeGroup.value?.items.map(buildConfigCard) ?? []);
const activeGroupOverrideCount = computed(() => activeGroup.value?.items.filter(hasConfigOverride).length ?? 0);
const editingSchema = computed(() =>
  editingItem.value ? editorSchemaForItem(editingItem.value) : parseConfigSchema(),
);
const editorContainer = computed<EditorContainer>(() =>
  editingItem.value && shouldUseDrawerEditor(editingItem.value) ? 'drawer' : 'dialog',
);
const editorDialogVisible = computed({
  get: () => editorVisible.value && editorContainer.value === 'dialog',
  set: (visible: boolean) => {
    if (!visible && editorContainer.value === 'dialog') {
      closeEditor();
    }
  },
});
const editorDrawerVisible = computed({
  get: () => editorVisible.value && editorContainer.value === 'drawer',
  set: (visible: boolean) => {
    if (!visible && editorContainer.value === 'drawer') {
      closeEditor();
    }
  },
});
const editorTitle = computed(() =>
  editingItem.value ? t('systemConfig.list.editorTitle', { title: configTitle(editingItem.value) }) : '',
);
const editorPreview = computed(() => formatJsonValue(editorForm.value) || t('systemConfig.list.emptyValue'));

onMounted(refreshConfigs);

watch(filteredDomains, (nextDomains) => {
  const visibleGroups = nextDomains.flatMap((domain) => domain.groups);
  if (!visibleGroups.some((group) => group.key === activeGroupKey.value)) {
    activeGroupKey.value = visibleGroups[0]?.key ?? '';
  }
  if (normalizedGroupSearchKeyword.value) {
    expandedDomainKeys.value = nextDomains.map((domain) => domain.key);
  }
});

async function refreshConfigs() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getSystemConfigs();
    items.value = response.items ?? [];
    if (!activeGroupKey.value || !groupedConfigs.value.some((group) => group.key === activeGroupKey.value)) {
      activeGroupKey.value = groupedConfigs.value[0]?.key ?? '';
    }
    expandedDomainKeys.value = domains.value.map((domain) => domain.key);
  } catch (error) {
    errorMessage.value = readableError(error, t('systemConfig.list.loadError'));
  } finally {
    loading.value = false;
  }
}

function handleTreeActive(value: TreeNodeValue[]) {
  const selected = String(value[0] ?? '');
  if (groupedConfigs.value.some((group) => group.key === selected)) {
    activeGroupKey.value = selected;
  }
}

function buildGroupSearchText(group: ConfigGroup, item: SystemConfigItem, previousSearchText = '') {
  return normalizeSearchText(
    [
      previousSearchText,
      group.label,
      group.description,
      group.key,
      group.domainKey,
      item.key,
      item.module,
      item.group,
      item.domain,
      configTitle(item),
      configDescription(item),
      item.tags?.join(' '),
    ].join(' '),
  );
}

function normalizeSearchText(value: string) {
  return value.trim().toLocaleLowerCase();
}

function openEditor(item: SystemConfigItem) {
  editingItem.value = item;
  editorForm.value = initialEditorValue(item);
  editorVisible.value = true;
}

function closeEditor() {
  editorVisible.value = false;
  editingItem.value = null;
}

async function saveEditor() {
  if (!editingItem.value) {
    return;
  }

  if (!editingItem.value.sensitive || editorForm.value !== null) {
    const validation = validateConfigEditorValue(editingSchema.value, editorForm.value, editingItem.value.type);
    if (!validation.valid) {
      MessagePlugin.error(schemaValidationMessage(validation.issues[0]));
      return;
    }
  }

  saving.value = true;
  try {
    const updated = await updateSystemConfig(editingItem.value.key, { value: editorForm.value });
    upsertConfig(updated);
    closeEditor();
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
    return emptySensitiveEditorValue(item.type);
  }

  return (
    parseJsonValue(item.override_value) ?? parseJsonValue(item.effective_value) ?? parseJsonValue(item.default_value)
  );
}

function emptySensitiveEditorValue(type: SystemConfigItem['type']) {
  switch (type) {
    case 'boolean':
      return false;
    case 'object':
      return {};
    case 'array':
      return [];
    case 'number':
    case 'integer':
      return null;
    default:
      return '';
  }
}

function editorSchemaForItem(item: SystemConfigItem) {
  const parsed = parseConfigSchema(item.config_schema);
  return {
    ...parsed,
    type: parsed.type || item.type,
    title: parsed.title || item.title || undefined,
    description: parsed.description || item.description || undefined,
    xI18n: {
      ...(parsed.xI18n ?? {}),
      titleKey: parsed.xI18n?.titleKey || item.title_key || undefined,
      descriptionKey: parsed.xI18n?.descriptionKey || item.description_key || undefined,
    },
  };
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

function groupDescription(item: SystemConfigItem) {
  return resolveI18nText(item.group_description_key, item.group_description, t('systemConfig.list.noDescription'));
}

function domainLabel(item: SystemConfigItem) {
  return resolveI18nText(item.domain_key, item.domain_label, technicalDomainKey(item));
}

function configDomainKey(item: SystemConfigItem) {
  return item.domain?.trim() || item.module || 'uncategorized';
}

function technicalDomainKey(item: SystemConfigItem) {
  return item.domain?.trim() || t('systemConfig.domains.uncategorized');
}

function technicalGroupKey(item: SystemConfigItem) {
  return item.group || t('systemConfig.list.defaultGroup');
}

function configEffectiveSource(item: SystemConfigItem) {
  if (hasConfigOverride(item)) {
    return {
      label: t('systemConfig.list.status.override'),
      description: t('systemConfig.list.status.overrideDescription'),
      theme: 'primary' as const,
    };
  }

  return {
    label: t('systemConfig.list.status.default'),
    description: t('systemConfig.list.status.defaultDescription'),
    theme: 'default' as const,
  };
}

function hasConfigOverride(item: SystemConfigItem) {
  return item.has_override || item.status === 'modified';
}

function isRuntimeHotConfig(item: SystemConfigItem) {
  return item.runtime_apply_mode === 'runtime_hot';
}

function configLastModifiedLabel(item: SystemConfigItem) {
  if (!hasConfigOverride(item)) {
    return t('systemConfig.list.lastModified.none');
  }

  const updatedAt = formatCompactDateTime(item.updated_at, locale);
  const userLabel = configUpdatedByLabel(item);
  return t('systemConfig.list.lastModified.value', { user: userLabel, time: updatedAt });
}

function configUpdatedByLabel(item: SystemConfigItem) {
  const username = item.updated_by_username?.trim();
  if (username) {
    return username;
  }

  if (item.updated_by_user_id !== undefined && item.updated_by_user_id !== null) {
    return t('systemConfig.list.lastModified.userId', { id: item.updated_by_user_id });
  }

  return t('systemConfig.list.lastModified.unknownUser');
}

function buildConfigCard(item: SystemConfigItem): ConfigCardVM {
  const schema = editorSchemaForItem(item);
  const fields = getConfigSchemaFields(schema);
  return {
    item,
    key: item.key,
    title: configTitle(item),
    description: configDescription(item),
    effectiveSource: configEffectiveSource(item),
    sensitive: item.sensitive,
    runtimeHot: isRuntimeHotConfig(item),
    restartRequired: item.restart_required,
    auditLabel: configLastModifiedLabel(item),
    valueSections: [
      buildValueSection(item, 'effective_value', t('systemConfig.list.values.current'), schema, fields),
      buildValueSection(item, 'default_value', t('systemConfig.list.values.default'), schema, fields),
    ],
    advanced: {
      currentJson: jsonPreviewFromRaw(item.effective_value),
      defaultJson: jsonPreviewFromRaw(item.default_value),
      schemaSummary: schemaSummary(item, schema, fields),
    },
    canReset: hasConfigOverride(item),
  };
}

function buildValueSection(
  item: SystemConfigItem,
  field: ConfigValueField,
  title: string,
  schema: ConfigSchema,
  fields: ConfigSchemaField[],
): ConfigValueSection {
  if (item.masked) {
    const maskedValue = item.masked_placeholder || '*****';
    return {
      key: field,
      title,
      rows: [{ key: field, label: title, rawValue: maskedValue }],
      extraRows: [],
    };
  }

  const parsed = parseJsonValue(item[field]);
  const rows = isJsonRecord(parsed) && fields.length > 0 ? structuredValueRows(parsed, fields) : [];
  const fallbackRows =
    rows.length > 0
      ? rows
      : [
          {
            key: field,
            label: schema.title
              ? resolveI18nText(schema.xI18n?.titleKey, schema.title, configTitle(item))
              : configTitle(item),
            rawValue: parsed,
            schema,
          },
        ];

  return {
    key: field,
    title,
    rows: fallbackRows.slice(0, CORE_VALUE_ROW_LIMIT),
    extraRows: fallbackRows.slice(CORE_VALUE_ROW_LIMIT),
  };
}

function structuredValueRows(value: Record<string, unknown>, fields: ConfigSchemaField[]): ConfigValueRow[] {
  return fields.map((field) => {
    return {
      key: field.key,
      label: schemaFieldTitle(field),
      rawValue: value[field.key],
      schema: field.schema,
      unit: schemaFieldUnit(field),
    };
  });
}

function booleanStateLabel(value: boolean) {
  return value ? t('systemConfig.list.boolean.enabled') : t('systemConfig.list.boolean.disabled');
}

function jsonPreviewFromRaw(raw?: string | null) {
  const parsed = parseJsonValue(raw);
  return parsed === undefined ? '' : formatJsonValue(parsed);
}

function schemaSummary(item: SystemConfigItem, schema: ConfigSchema, fields: ConfigSchemaField[]) {
  const summary = [t('systemConfig.list.advanced.schemaType', { type: schema.type || item.type })];
  if (fields.length > 0) {
    summary.push(t('systemConfig.list.advanced.schemaFieldCount', { count: fields.length }));
  }
  const requiredCount = fields.filter((field) => field.required).length;
  if (requiredCount > 0) {
    summary.push(t('systemConfig.list.advanced.schemaRequiredCount', { count: requiredCount }));
  }
  if (schema.additionalProperties === false) {
    summary.push(t('systemConfig.list.advanced.schemaNoAdditionalProperties'));
  }
  return summary;
}

function shouldUseDrawerEditor(item: SystemConfigItem) {
  const schema = editorSchemaForItem(item);
  return configEditorContainer(schema, getConfigSchemaFields(schema)) === 'drawer';
}

async function copyConfigKey(key: string) {
  let copied = false;
  try {
    copied = await copyText(key);
  } catch {
    copied = false;
  }

  if (copied) {
    MessagePlugin.success(t('systemConfig.list.advanced.copySuccess'));
    return;
  }

  MessagePlugin.error(t('systemConfig.list.advanced.copyError'));
}

function schemaDescription(schema: ConfigSchemaProperty) {
  return resolveI18nText(schema.xI18n?.descriptionKey, schema.description, '');
}

function schemaOptionDisplayLabel(schema: ConfigSchemaProperty, value: unknown) {
  const optionText = schema.enumLabels?.[String(value)];
  return resolveI18nText(optionText?.labelKey, optionText?.label, String(value));
}

function schemaOptionDescription(schema: ConfigSchemaProperty, value: unknown) {
  const optionText = schema.enumLabels?.[String(value)];
  return resolveI18nText(optionText?.descriptionKey, optionText?.description, '');
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
  return schemaOptionDisplayLabel(field.schema, option);
}

function schemaValidationMessage(issue?: {
  constraint?: string;
  expected?: unknown;
  maximum?: number;
  minimum?: number;
}) {
  if (!issue) {
    return t('systemConfig.list.schema.invalidValue');
  }
  switch (issue.constraint) {
    case 'enum':
      return t('systemConfig.list.schema.invalidEnum');
    case 'minimum':
      return t('systemConfig.list.schema.belowMinimum', { minimum: issue.minimum });
    case 'maximum':
      return t('systemConfig.list.schema.aboveMaximum', { maximum: issue.maximum });
    case 'minLength':
      return t('systemConfig.list.schema.belowMinLength', { minLength: issue.expected });
    case 'maxLength':
      return t('systemConfig.list.schema.aboveMaxLength', { maxLength: issue.expected });
    default:
      return t('systemConfig.list.schema.invalidValue');
  }
}

function resolveI18nText(key?: string, fallback?: string, rawFallback = '') {
  const localized = resolveLocaleMessage(key);
  if (localized) {
    return localized;
  }

  return fallback || rawFallback;
}

function resolveLocaleMessage(key?: string) {
  const trimmed = key?.trim();
  if (!trimmed) {
    return '';
  }

  const messages = localeMessages();
  const exact = messages[trimmed];
  if (typeof exact === 'string' && exact.trim()) {
    return exact;
  }

  if (te(trimmed)) {
    const resolved = t(trimmed);
    if (resolved && resolved !== trimmed) {
      return resolved;
    }
  }

  const nested = trimmed.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') {
      return undefined;
    }

    return (current as Record<string, unknown>)[segment];
  }, messages);

  return typeof nested === 'string' && nested.trim() ? nested : '';
}

function localeMessages() {
  return getLocaleMessage(locale.value) as Record<string, unknown>;
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
  height: calc(100vh - 188px);
  min-height: 560px;
  min-width: 0;
  overflow: hidden;
}

.system-config-page__alert {
  flex: 0 0 auto;
}

.system-config-content__head h2,
.system-config-item h3 {
  margin: 0;
}

.system-config-content__head p,
.system-config-item p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
}

.system-config-workspace,
.system-config-workspace :deep(.t-loading__parent) {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}

.system-config-layout {
  align-items: stretch;
  display: grid;
  flex: 1 1 auto;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.system-config-groups {
  align-self: stretch;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  height: 100%;
  max-height: 100%;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: var(--graft-density-gap-12);
  scrollbar-gutter: stable;
}

.system-config-groups__search {
  flex: 0 0 auto;
}

.system-config-groups :deep(.t-tree) {
  background: transparent;
  min-height: 0;
}

.system-config-scrollbar {
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-width: thin;
}

.system-config-scrollbar::-webkit-scrollbar {
  background: transparent;
  width: 8px;
}

.system-config-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.system-config-scrollbar::-webkit-scrollbar-thumb {
  background-clip: content-box;
  background-color: var(--td-scrollbar-color);
  border: 2px solid transparent;
  border-radius: 6px;
}

.system-config-tree-node {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  line-height: 1.4;
  min-width: 0;
}

.system-config-tree-node small {
  color: var(--td-text-color-secondary);
}

.system-config-content {
  align-self: stretch;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  height: 100%;
  max-height: 100%;
  min-height: 0;
  min-width: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: var(--graft-density-gap-4);
  scrollbar-gutter: stable;
}

.system-config-content__head {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
  position: sticky;
  top: 0;
  z-index: 2;
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
  padding-bottom: var(--graft-density-gap-24);
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
.system-config-advanced__section {
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
.system-config-advanced__section > span,
.system-config-advanced__section-head > span {
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

.system-config-value__display {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.system-config-value__info {
  align-items: center;
  appearance: none;
  background: transparent;
  border: 0;
  color: var(--td-text-color-placeholder);
  cursor: help;
  display: inline-flex;
  flex: 0 0 auto;
  height: 18px;
  justify-content: center;
  padding: 0;
  width: 18px;
}

.system-config-value__info:hover,
.system-config-value__info:focus-visible {
  color: var(--td-brand-color);
  outline: none;
}

.system-config-value__rows strong,
.system-config-advanced code {
  overflow-wrap: anywhere;
}

.system-config-more-fields :deep(.t-collapse-panel__content),
.system-config-advanced :deep(.t-collapse-panel__content),
.system-config-editor__preview :deep(.t-collapse-panel__content) {
  padding: 0;
}

.system-config-advanced__grid,
.system-config-advanced__section,
.system-config-editor--drawer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.system-config-advanced__section-head,
.system-config-editor__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.system-config-advanced__section ul {
  margin: 0;
  padding-left: var(--graft-density-gap-18);
}

.system-config-advanced__section li {
  color: var(--td-text-color-secondary);
  line-height: 1.6;
}

.system-config-advanced pre,
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

.system-config-advanced pre {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
}

.system-config-editor__preview pre {
  background: var(--td-bg-color-page);
}

.system-config-editor__actions {
  border-top: 1px solid var(--td-border-level-1-color);
  justify-content: flex-end;
  padding-top: var(--graft-density-gap-12);
}

@media (width <= 900px) {
  .system-config-page {
    height: auto;
    min-height: 0;
    overflow: visible;
  }

  .system-config-workspace,
  .system-config-workspace :deep(.t-loading__parent) {
    height: auto;
    overflow: visible;
  }

  .system-config-layout,
  .system-config-summary,
  .system-config-values {
    display: flex;
    flex-direction: column;
    overflow: visible;
  }

  .system-config-groups,
  .system-config-content {
    height: auto;
    max-height: none;
    overflow: visible;
    padding-right: 0;
    width: 100%;
  }

  .system-config-content__head {
    position: static;
  }

  .system-config-item__actions {
    flex-flow: row wrap;
  }
}
</style>
