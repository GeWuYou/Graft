<template>
  <section class="app-log-filters">
    <t-form :data="modelValue" layout="inline" label-align="top" @submit.prevent="$emit('search')">
      <t-form-item :label="t('appLog.filters.keyword')">
        <t-input
          :value="modelValue.keyword"
          clearable
          :placeholder="t('appLog.page.searchPlaceholder')"
          @change="updateField('keyword', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.occurredRange')">
        <t-date-range-picker
          :value="modelValue.occurredRange"
          clearable
          enable-time-picker
          :placeholder="[t('appLog.filters.occurredRange'), t('appLog.filters.occurredRange')]"
          @change="updateField('occurredRange', normalizeRange($event))"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.severity')">
        <t-select
          :value="modelValue.severity"
          clearable
          :placeholder="t('appLog.filters.allSeverity')"
          :options="severityOptions"
          @change="updateField('severity', normalizeSeverity($event))"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.component')">
        <t-input
          :value="modelValue.component"
          clearable
          :placeholder="t('appLog.filters.component')"
          @change="updateField('component', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.operation')">
        <t-input
          :value="modelValue.operation"
          clearable
          :placeholder="t('appLog.filters.operation')"
          @change="updateField('operation', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.requestId')">
        <t-input
          :value="modelValue.requestId"
          clearable
          :placeholder="t('appLog.filters.requestId')"
          @change="updateField('requestId', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.traceId')">
        <t-input
          :value="modelValue.traceId"
          clearable
          :placeholder="t('appLog.filters.traceId')"
          @change="updateField('traceId', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.message')">
        <t-input
          :value="modelValue.message"
          clearable
          :placeholder="t('appLog.filters.message')"
          @change="updateField('message', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item :label="t('appLog.filters.error')">
        <t-input
          :value="modelValue.error"
          clearable
          :placeholder="t('appLog.filters.error')"
          @change="updateField('error', String($event || ''))"
          @enter="$emit('search')"
        />
      </t-form-item>
      <t-form-item class="app-log-filters__actions">
        <t-button theme="primary" type="submit" :loading="loading">{{ t('appLog.actions.search') }}</t-button>
        <t-button theme="default" variant="outline" @click="$emit('reset')">{{ t('appLog.actions.reset') }}</t-button>
      </t-form-item>
    </t-form>
  </section>
</template>
<script setup lang="ts">
import type { SelectValue } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { AppLogFilterState, AppLogSeverity } from '../types/app-log';

const props = defineProps<{
  loading?: boolean;
  modelValue: AppLogFilterState;
}>();

const emit = defineEmits<{
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:modelValue', value: AppLogFilterState): void;
}>();

const { t } = useI18n();

const severityOptions = computed(() =>
  (['debug', 'info', 'warn', 'error'] satisfies AppLogSeverity[]).map((value) => ({
    label: value.toUpperCase(),
    value,
  })),
);

function updateField<Key extends keyof AppLogFilterState>(key: Key, value: AppLogFilterState[Key]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function normalizeRange(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

function normalizeSeverity(value: SelectValue): AppLogFilterState['severity'] {
  return value === 'debug' || value === 'info' || value === 'warn' || value === 'error' ? value : '';
}
</script>
<style scoped lang="less">
.app-log-filters {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  padding: 16px;
}

.app-log-filters :deep(.t-form__item) {
  min-width: 210px;
}

.app-log-filters :deep(.t-date-range-picker) {
  min-width: 320px;
}

.app-log-filters__actions {
  align-items: end;
}

@media (width <= 768px) {
  .app-log-filters :deep(.t-form__item),
  .app-log-filters :deep(.t-date-range-picker) {
    min-width: 100%;
  }
}
</style>
