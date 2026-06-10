<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <template v-if="objectFields.length > 0">
    <t-form-item
      v-for="field in objectFields"
      :key="field.key"
      :label="fieldTitle(field)"
      :name="fieldName(field.key)"
      :help="fieldDescription(field)"
      :required-mark="field.required"
    >
      <template v-if="field.schema.enum">
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
      <t-input-number
        v-else-if="field.schema.type === 'integer' || field.schema.type === 'number'"
        :model-value="numberValue(objectValue[field.key])"
        :min="field.schema.minimum"
        :max="field.schema.maximum"
        :decimal-places="field.schema.type === 'integer' ? 0 : undefined"
        :placeholder="fieldPlaceholder(field, labels.numberPlaceholder)"
        :suffix="fieldUnit(field)"
        :disabled="disabled"
        @change="(value) => updateObjectField(field.key, value)"
      />
      <t-switch
        v-else-if="field.schema.type === 'boolean'"
        :model-value="Boolean(objectValue[field.key])"
        :disabled="disabled"
        @change="(value) => updateObjectField(field.key, value)"
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
    <template v-if="rootSchema.enum">
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
    <t-input-number
      v-else-if="rootSchema.type === 'integer' || rootSchema.type === 'number'"
      :model-value="numberValue(modelValue)"
      :min="rootSchema.minimum"
      :max="rootSchema.maximum"
      :decimal-places="rootSchema.type === 'integer' ? 0 : undefined"
      :placeholder="rootPlaceholder(labels.numberPlaceholder)"
      :suffix="rootUnit"
      :disabled="disabled"
      @change="(value) => emit('update:modelValue', value)"
    />
    <t-switch
      v-else-if="rootSchema.type === 'boolean'"
      :model-value="Boolean(modelValue)"
      :disabled="disabled"
      @change="(value) => emit('update:modelValue', value)"
    />
    <t-textarea
      v-else-if="rootSchema.type === 'object' || rootSchema.type === 'array'"
      :model-value="formatJsonValue(modelValue)"
      class="json-schema-value-fields__textarea"
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
import { computed, ref } from 'vue';

import type { ConfigSchema, ConfigSchemaField } from './config-schema';
import { getConfigSchemaFields } from './config-schema';
import { formatJsonValue, isJsonRecord, type JsonRecord, parseJsonValue } from './json';

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    fieldPrefix?: string;
    labels: {
      invalidJson: string;
      jsonPlaceholder: string;
      numberPlaceholder: string;
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
const objectFields = computed(() =>
  props.rootSchema.type === 'object' ? getConfigSchemaFields(props.rootSchema) : [],
);
const objectValue = computed<JsonRecord>(() => (isJsonRecord(props.modelValue) ? props.modelValue : {}));
const rootField = computed<ConfigSchemaField>(() => ({
  key: 'value',
  schema: props.rootSchema,
  required: false,
}));
const rootLabel = computed(
  () => props.titleResolver?.(rootField.value) || props.rootSchema.title || props.labels.value,
);
const rootHelp = computed(() => fieldDescription(rootField.value));
const rootUnit = computed(() => fieldUnit(rootField.value));

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
</script>
<style scoped>
.json-schema-value-fields__textarea {
  width: 100%;
}

:deep(.json-schema-value-fields__textarea.t-textarea),
.json-schema-value-fields__textarea :deep(.t-textarea__inner) {
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  width: 100%;
}
</style>
