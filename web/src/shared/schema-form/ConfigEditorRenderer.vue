<template>
  <template v-if="objectFields.length > 0">
    <t-form-item
      v-for="field in objectFields"
      :key="field.key"
      :label="fieldTitle(field)"
      :name="fieldName(field.key)"
      :help="fieldDescription(field)"
      :required-mark="field.required"
      :status="jsonFieldErrors[field.key] ? 'error' : undefined"
      :tips="jsonFieldErrors[field.key]"
    >
      <template v-if="rendererKind(field) === 'select'">
        <t-select
          :model-value="selectValue(objectValue[field.key])"
          :placeholder="fieldPlaceholder(field, labels.selectPlaceholder)"
          :disabled="disabled"
          clearable
          @change="(value) => updateObjectField(field.key, value)"
        >
          <t-option
            v-for="option in field.schema.enum"
            :key="String(option)"
            :value="option"
            :label="optionLabel(field, option)"
          />
        </t-select>
      </template>
      <div v-else-if="rendererKind(field) === 'input-number'" class="config-editor-renderer__number-row">
        <t-input-number
          class="config-editor-renderer__number-input"
          :model-value="numberValue(objectValue[field.key])"
          :min="field.schema.minimum"
          :max="field.schema.maximum"
          :decimal-places="field.schema.type === 'integer' ? 0 : undefined"
          :placeholder="fieldPlaceholder(field, labels.numberPlaceholder)"
          :disabled="disabled"
          align="center"
          theme="row"
          @change="(value) => updateObjectField(field.key, value)"
        />
        <span v-if="fieldUnit(field)" class="config-editor-renderer__number-unit">
          {{ fieldUnit(field) }}
        </span>
      </div>
      <t-switch
        v-else-if="rendererKind(field) === 'switch'"
        :model-value="Boolean(objectValue[field.key])"
        :disabled="disabled"
        @change="(value) => updateObjectField(field.key, value)"
      />
      <t-tag-input
        v-else-if="rendererKind(field) === 'string-array-list'"
        :value="stringListValue(objectValue[field.key])"
        :placeholder="fieldPlaceholder(field, labels.stringPlaceholder)"
        :disabled="disabled"
        clearable
        excess-tags-display-type="break-line"
        @change="(value) => updateObjectField(field.key, serializeStringList(value))"
      />
      <div v-else-if="rendererKind(field) === 'workspace-tooltip-rule-list'" class="config-editor-renderer__rule-list">
        <div
          v-for="(rule, index) in workspaceTooltipRuleListValue(objectValue[field.key])"
          :key="`${field.key}-${index}`"
          class="config-editor-renderer__rule-card"
        >
          <div class="config-editor-renderer__rule-card-head">
            <strong>{{ `#${index + 1}` }}</strong>
            <t-space size="small">
              <t-button
                size="small"
                variant="text"
                :disabled="disabled || index === 0"
                @click="moveRule(field.key, index, -1)"
              >
                {{ labels.ruleUpAction }}
              </t-button>
              <t-button
                size="small"
                variant="text"
                :disabled="disabled || index === workspaceTooltipRuleListValue(objectValue[field.key]).length - 1"
                @click="moveRule(field.key, index, 1)"
              >
                {{ labels.ruleDownAction }}
              </t-button>
              <t-button
                size="small"
                theme="danger"
                variant="text"
                :disabled="disabled"
                @click="removeRule(field.key, index)"
              >
                {{ labels.ruleRemoveAction }}
              </t-button>
            </t-space>
          </div>
          <t-input
            :model-value="rule.pattern"
            :placeholder="fieldPlaceholder(field, labels.rulePatternPlaceholder)"
            :disabled="disabled"
            clearable
            @change="(value) => updateRuleField(field.key, index, 'pattern', String(value ?? ''))"
          />
          <t-input
            :model-value="rule.tooltip"
            :placeholder="labels.ruleTooltipPlaceholder"
            :disabled="disabled"
            clearable
            @change="(value) => updateRuleField(field.key, index, 'tooltip', String(value ?? ''))"
          />
          <div class="config-editor-renderer__rule-enabled">
            <span>{{ labels.ruleEnabledLabel }}</span>
            <t-switch
              :model-value="rule.enabled"
              :disabled="disabled"
              @change="(value) => updateRuleField(field.key, index, 'enabled', Boolean(value))"
            />
          </div>
        </div>
        <t-button size="small" variant="outline" :disabled="disabled" @click="appendRule(field.key)">
          {{ labels.ruleAddAction }}
        </t-button>
      </div>
      <t-textarea
        v-else-if="rendererKind(field) === 'json-textarea'"
        :model-value="formatJsonValue(objectValue[field.key])"
        class="config-editor-renderer__textarea"
        :autosize="{ minRows: 5, maxRows: 10 }"
        :placeholder="fieldPlaceholder(field, labels.jsonPlaceholder)"
        :disabled="disabled"
        @change="(value) => handleObjectJsonChange(field.key, value)"
      />
      <t-input
        v-else
        :model-value="stringValue(objectValue[field.key])"
        :minlength="field.schema.minLength"
        :maxlength="field.schema.maxLength"
        :placeholder="fieldPlaceholder(field, labels.stringPlaceholder)"
        :disabled="disabled"
        clearable
        @change="(value) => updateObjectField(field.key, value)"
      />
    </t-form-item>
  </template>

  <t-form-item
    v-else
    :label="rootLabel"
    name="value"
    :help="rootHelp"
    :status="jsonError ? 'error' : undefined"
    :tips="jsonError"
  >
    <template v-if="rootRendererKind === 'select'">
      <t-select
        :model-value="selectValue(modelValue)"
        :placeholder="rootPlaceholder(labels.selectPlaceholder)"
        :disabled="disabled"
        clearable
        @change="(value) => emit('update:modelValue', value)"
      >
        <t-option
          v-for="option in rootSchema.enum"
          :key="String(option)"
          :value="option"
          :label="optionLabel(rootField, option)"
        />
      </t-select>
    </template>
    <div v-else-if="rootRendererKind === 'input-number'" class="config-editor-renderer__number-row">
      <t-input-number
        class="config-editor-renderer__number-input"
        :model-value="numberValue(modelValue)"
        :min="rootSchema.minimum"
        :max="rootSchema.maximum"
        :decimal-places="rootSchema.type === 'integer' ? 0 : undefined"
        :placeholder="rootPlaceholder(labels.numberPlaceholder)"
        :disabled="disabled"
        align="center"
        theme="row"
        @change="(value) => emit('update:modelValue', value)"
      />
      <span v-if="rootUnit" class="config-editor-renderer__number-unit">
        {{ rootUnit }}
      </span>
    </div>
    <t-switch
      v-else-if="rootRendererKind === 'switch'"
      :model-value="Boolean(modelValue)"
      :disabled="disabled"
      @change="(value) => emit('update:modelValue', value)"
    />
    <t-tag-input
      v-else-if="rootRendererKind === 'string-array-list'"
      :value="stringListValue(modelValue)"
      :placeholder="rootPlaceholder(labels.stringPlaceholder)"
      :disabled="disabled"
      clearable
      excess-tags-display-type="break-line"
      @change="(value) => emit('update:modelValue', serializeStringList(value))"
    />
    <div v-else-if="rootRendererKind === 'workspace-tooltip-rule-list'" class="config-editor-renderer__rule-list">
      <div
        v-for="(rule, index) in workspaceTooltipRuleListValue(modelValue)"
        :key="`root-${index}`"
        class="config-editor-renderer__rule-card"
      >
        <div class="config-editor-renderer__rule-card-head">
          <strong>{{ `#${index + 1}` }}</strong>
          <t-space size="small">
            <t-button size="small" variant="text" :disabled="disabled || index === 0" @click="moveRootRule(index, -1)">
              {{ labels.ruleUpAction }}
            </t-button>
            <t-button
              size="small"
              variant="text"
              :disabled="disabled || index === workspaceTooltipRuleListValue(modelValue).length - 1"
              @click="moveRootRule(index, 1)"
            >
              {{ labels.ruleDownAction }}
            </t-button>
            <t-button size="small" theme="danger" variant="text" :disabled="disabled" @click="removeRootRule(index)">{{
              labels.ruleRemoveAction
            }}</t-button>
          </t-space>
        </div>
        <t-input
          :model-value="rule.pattern"
          :placeholder="rootPlaceholder(labels.rulePatternPlaceholder)"
          :disabled="disabled"
          clearable
          @change="(value) => updateRootRuleField(index, 'pattern', String(value ?? ''))"
        />
        <t-input
          :model-value="rule.tooltip"
          :placeholder="labels.ruleTooltipPlaceholder"
          :disabled="disabled"
          clearable
          @change="(value) => updateRootRuleField(index, 'tooltip', String(value ?? ''))"
        />
        <div class="config-editor-renderer__rule-enabled">
          <span>{{ labels.ruleEnabledLabel }}</span>
          <t-switch
            :model-value="rule.enabled"
            :disabled="disabled"
            @change="(value) => updateRootRuleField(index, 'enabled', Boolean(value))"
          />
        </div>
      </div>
      <t-button size="small" variant="outline" :disabled="disabled" @click="appendRootRule">
        {{ labels.ruleAddAction }}
      </t-button>
    </div>
    <t-textarea
      v-else-if="rootRendererKind === 'json-textarea'"
      :model-value="formatJsonValue(modelValue)"
      class="config-editor-renderer__textarea"
      :autosize="{ minRows: 5, maxRows: 10 }"
      :placeholder="rootPlaceholder(labels.jsonPlaceholder)"
      :disabled="disabled"
      @change="handleJsonChange"
    />
    <t-input
      v-else
      :model-value="stringValue(modelValue)"
      :minlength="rootSchema.minLength"
      :maxlength="rootSchema.maxLength"
      :placeholder="rootPlaceholder(labels.stringPlaceholder)"
      :disabled="disabled"
      clearable
      @change="(value) => emit('update:modelValue', value)"
    />
  </t-form-item>
</template>
<script setup lang="ts">
import { computed, reactive, ref } from 'vue';

import type { ConfigFieldType, ConfigSchema, ConfigSchemaField } from './config-schema';
import { getConfigSchemaFields } from './config-schema';
import { configFieldRenderer } from './field-renderer';
import { formatJsonValue, isJsonRecord, type JsonRecord, parseJsonValue } from './json';

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    fallbackType?: ConfigFieldType | null;
    fieldPrefix?: string;
    labels: {
      invalidJson: string;
      jsonPlaceholder: string;
      numberPlaceholder: string;
      ruleAddAction: string;
      ruleDownAction: string;
      ruleEnabledLabel: string;
      rulePatternPlaceholder: string;
      ruleRemoveAction: string;
      ruleTooltipPlaceholder: string;
      ruleUpAction: string;
      selectPlaceholder: string;
      stringPlaceholder: string;
      value: string;
    };
    modelValue: unknown;
    rootSchema: ConfigSchema;
    titleResolver?: (field: ConfigSchemaField) => string;
    descriptionResolver?: (field: ConfigSchemaField) => string;
    optionLabelResolver?: (field: ConfigSchemaField, option: string | number | boolean) => string;
    placeholderResolver?: (field: ConfigSchemaField) => string;
    unitResolver?: (field: ConfigSchemaField) => string;
  }>(),
  {
    disabled: false,
    fallbackType: undefined,
    fieldPrefix: 'value',
    titleResolver: undefined,
    descriptionResolver: undefined,
    optionLabelResolver: undefined,
    placeholderResolver: undefined,
    unitResolver: undefined,
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: unknown];
}>();

const jsonError = ref('');
const jsonFieldErrors = reactive<Record<string, string>>({});
const objectFields = computed(() =>
  props.rootSchema.type === 'object' ? getConfigSchemaFields(props.rootSchema) : [],
);
const objectValue = computed<JsonRecord>(() => (isJsonRecord(props.modelValue) ? props.modelValue : {}));
const rootField = computed<ConfigSchemaField>(() => ({
  key: 'value',
  schema: props.rootSchema,
  required: false,
}));
const rootRendererKind = computed(() => configFieldRenderer(props.rootSchema, props.fallbackType));
const rootLabel = computed(
  () => props.titleResolver?.(rootField.value) || props.rootSchema.title || props.labels.value,
);
const rootHelp = computed(() => fieldDescription(rootField.value));
const rootUnit = computed(() => fieldUnit(rootField.value));

type WorkspaceTooltipRule = {
  enabled: boolean;
  pattern: string;
  tooltip: string;
};

function rendererKind(field: ConfigSchemaField) {
  return configFieldRenderer(field.schema);
}

function fieldName(key: string) {
  return `${props.fieldPrefix}.${key}`;
}

function fieldTitle(field: ConfigSchemaField) {
  return props.titleResolver?.(field) || field.schema.title || field.key;
}

function fieldDescription(field: ConfigSchemaField) {
  return props.descriptionResolver?.(field) || field.schema.description || '';
}

function fieldPlaceholder(field: ConfigSchemaField, fallback: string) {
  return props.placeholderResolver?.(field) || field.schema.placeholder || fallback;
}

function fieldUnit(field: ConfigSchemaField) {
  return props.unitResolver?.(field) || undefined;
}

function rootPlaceholder(fallback: string) {
  return fieldPlaceholder(rootField.value, fallback);
}

function optionLabel(field: ConfigSchemaField, option: string | number | boolean) {
  return (
    props.optionLabelResolver?.(field, option) || field.schema.enumLabels?.[String(option)]?.label || String(option)
  );
}

function updateObjectField(key: string, value: unknown) {
  jsonFieldErrors[key] = '';
  emit('update:modelValue', {
    ...objectValue.value,
    [key]: value,
  });
}

function numberValue(value: unknown) {
  return typeof value === 'number' ? value : undefined;
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : value === undefined || value === null ? '' : String(value);
}

function selectValue(value: unknown) {
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean' ? value : undefined;
}

function stringListValue(value: unknown) {
  if (typeof value !== 'string') {
    return [];
  }
  const parsed = parseJsonValue(value);
  if (!Array.isArray(parsed)) {
    return [];
  }
  return parsed
    .filter((item): item is string | number => typeof item === 'string' || typeof item === 'number')
    .map((item) => String(item));
}

function serializeStringList(value: unknown) {
  if (!Array.isArray(value)) {
    return '[]';
  }
  return JSON.stringify(value.map((item) => String(item).trim()).filter((item) => item.length > 0));
}

function workspaceTooltipRuleListValue(value: unknown) {
  const parsed = typeof value === 'string' ? parseJsonValue(value) : value;
  if (!Array.isArray(parsed)) {
    return [] as WorkspaceTooltipRule[];
  }
  return parsed
    .filter((item): item is JsonRecord => isJsonRecord(item))
    .map((item) => ({
      enabled: item.enabled !== false,
      pattern: typeof item.pattern === 'string' ? item.pattern : '',
      tooltip: typeof item.tooltip === 'string' ? item.tooltip : '',
    }));
}

function serializeWorkspaceTooltipRules(value: WorkspaceTooltipRule[]) {
  return JSON.stringify(
    value.map((item) => ({
      enabled: item.enabled !== false,
      pattern: item.pattern.trim(),
      tooltip: item.tooltip.trim(),
    })),
  );
}

function nextWorkspaceTooltipRules(value: unknown, mutate: (items: WorkspaceTooltipRule[]) => WorkspaceTooltipRule[]) {
  return serializeWorkspaceTooltipRules(mutate([...workspaceTooltipRuleListValue(value)]));
}

function appendRule(key: string) {
  updateObjectField(
    key,
    nextWorkspaceTooltipRules(objectValue.value[key], (items) => [
      ...items,
      { enabled: true, pattern: '', tooltip: '' },
    ]),
  );
}

function removeRule(key: string, index: number) {
  updateObjectField(
    key,
    nextWorkspaceTooltipRules(objectValue.value[key], (items) => items.filter((_, itemIndex) => itemIndex !== index)),
  );
}

function moveRule(key: string, index: number, offset: -1 | 1) {
  updateObjectField(
    key,
    nextWorkspaceTooltipRules(objectValue.value[key], (items) => moveWorkspaceTooltipRule(items, index, offset)),
  );
}

function updateRuleField(key: string, index: number, field: keyof WorkspaceTooltipRule, value: string | boolean) {
  updateObjectField(
    key,
    nextWorkspaceTooltipRules(objectValue.value[key], (items) =>
      items.map((item, itemIndex) => (itemIndex === index ? { ...item, [field]: value } : item)),
    ),
  );
}

function appendRootRule() {
  emit(
    'update:modelValue',
    nextWorkspaceTooltipRules(props.modelValue, (items) => [...items, { enabled: true, pattern: '', tooltip: '' }]),
  );
}

function removeRootRule(index: number) {
  emit(
    'update:modelValue',
    nextWorkspaceTooltipRules(props.modelValue, (items) => items.filter((_, itemIndex) => itemIndex !== index)),
  );
}

function moveRootRule(index: number, offset: -1 | 1) {
  emit(
    'update:modelValue',
    nextWorkspaceTooltipRules(props.modelValue, (items) => moveWorkspaceTooltipRule(items, index, offset)),
  );
}

function updateRootRuleField(index: number, field: keyof WorkspaceTooltipRule, value: string | boolean) {
  emit(
    'update:modelValue',
    nextWorkspaceTooltipRules(props.modelValue, (items) =>
      items.map((item, itemIndex) => (itemIndex === index ? { ...item, [field]: value } : item)),
    ),
  );
}

function moveWorkspaceTooltipRule(items: WorkspaceTooltipRule[], index: number, offset: -1 | 1) {
  const targetIndex = index + offset;
  if (index < 0 || targetIndex < 0 || index >= items.length || targetIndex >= items.length) {
    return items;
  }
  const nextItems = [...items];
  const [current] = nextItems.splice(index, 1);
  nextItems.splice(targetIndex, 0, current);
  return nextItems;
}

function handleJsonChange(value: string | number) {
  const text = String(value ?? '');
  const parsed = parseJsonValue(text);
  if (parsed === undefined && text.trim()) {
    jsonError.value = props.labels.invalidJson;
    return;
  }

  jsonError.value = '';
  emit('update:modelValue', parsed);
}

function handleObjectJsonChange(key: string, value: string | number) {
  const text = String(value ?? '');
  const parsed = parseJsonValue(text);
  if (parsed === undefined && text.trim()) {
    jsonFieldErrors[key] = props.labels.invalidJson;
    return;
  }

  jsonFieldErrors[key] = '';
  updateObjectField(key, parsed);
}
</script>
<style scoped>
.config-editor-renderer__textarea {
  width: 100%;
}

.config-editor-renderer__rule-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.config-editor-renderer__rule-card {
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-12);
}

.config-editor-renderer__rule-card-head,
.config-editor-renderer__rule-enabled {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.config-editor-renderer__rule-enabled > span,
.config-editor-renderer__rule-card-head > strong {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.config-editor-renderer__number-row {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-xs);
  max-width: 240px;
}

.config-editor-renderer__number-input {
  flex: 0 0 120px;
  min-width: 120px;
  width: 120px;
}

.config-editor-renderer__number-input :deep(.t-input-number),
:deep(.config-editor-renderer__number-input.t-input-number) {
  width: 100%;
}

.config-editor-renderer__number-input :deep(.t-input__inner) {
  font-variant-numeric: tabular-nums;
  overflow: visible;
  text-align: center;
  text-overflow: clip;
  white-space: nowrap;
}

.config-editor-renderer__number-unit {
  color: var(--td-text-color-secondary);
  flex: 0 0 auto;
  line-height: var(--td-line-height-body-medium);
  white-space: nowrap;
}

:deep(.config-editor-renderer__textarea.t-textarea),
.config-editor-renderer__textarea :deep(.t-textarea__inner) {
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  width: 100%;
}
</style>
