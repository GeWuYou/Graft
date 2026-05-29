<template>
  <management-toolbar>
    <template #filters>
      <div class="audit-filters">
        <div class="audit-filters__row">
          <t-input
            :model-value="modelValue.keyword"
            class="audit-filters__keyword"
            clearable
            :placeholder="t('audit.logList.filters.keywordPlaceholder')"
            @update:model-value="updateField('keyword', $event)"
          />
          <t-input
            :model-value="modelValue.actor"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.actorPlaceholder')"
            @update:model-value="updateField('actor', $event)"
          />
          <t-select
            :model-value="modelValue.action"
            class="audit-filters__input"
            clearable
            :options="actionOptions"
            :placeholder="t('audit.logList.filters.actionPlaceholder')"
            @update:model-value="updateField('action', $event)"
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
        </div>
        <div v-if="advancedVisible" class="audit-filters__row">
          <t-input
            :model-value="modelValue.resource"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.resourcePlaceholder')"
            @update:model-value="updateField('resource', $event)"
          />
          <t-input
            :model-value="modelValue.resourceId"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.resourceIdPlaceholder')"
            @update:model-value="updateField('resourceId', $event)"
          />
          <t-select
            :model-value="modelValue.result"
            class="audit-filters__input"
            clearable
            :options="resultOptions"
            :placeholder="t('audit.logList.filters.resultPlaceholder')"
            @update:model-value="updateField('result', $event)"
          />
          <t-select
            :model-value="modelValue.riskLevel"
            class="audit-filters__input"
            clearable
            :options="riskOptions"
            :placeholder="t('audit.logList.filters.riskPlaceholder')"
            @update:model-value="updateField('riskLevel', $event)"
          />
        </div>
        <div v-if="advancedVisible" class="audit-filters__row">
          <t-input
            :model-value="modelValue.session"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.sessionPlaceholder')"
            @update:model-value="updateField('session', $event)"
          />
          <t-input
            :model-value="modelValue.requestId"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.requestIdPlaceholder')"
            @update:model-value="updateField('requestId', $event)"
          />
          <t-input
            :model-value="modelValue.traceId"
            class="audit-filters__input"
            clearable
            :placeholder="t('audit.logList.filters.traceIdPlaceholder')"
            @update:model-value="updateField('traceId', $event)"
          />
        </div>
      </div>
    </template>

    <template #actions>
      <t-space size="small" wrap>
        <t-button theme="default" variant="outline" @click="$emit('toggle-advanced')">
          {{ advancedVisible ? t('audit.logList.actions.hideAdvanced') : t('audit.logList.actions.showAdvanced') }}
        </t-button>
        <t-button theme="default" variant="outline" @click="$emit('reset')">
          {{ t('audit.logList.actions.reset') }}
        </t-button>
        <t-button theme="primary" :loading="loading" @click="$emit('search')">
          {{ t('audit.logList.actions.search') }}
        </t-button>
      </t-space>
    </template>
  </management-toolbar>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementToolbar } from '@/shared/components/management';

import type { AuditClientFilterState } from '../shared/presentation';

const props = defineProps<{
  advancedVisible: boolean;
  loading?: boolean;
  modelValue: AuditClientFilterState;
}>();

const emit = defineEmits<{
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'toggle-advanced'): void;
  (e: 'update:modelValue', value: AuditClientFilterState): void;
}>();

const { t } = useI18n();

const actionOptions = computed(() => [
  { label: t('audit.logList.filterOptions.allActions'), value: '' },
  { label: t('audit.logList.filterOptions.auth'), value: 'auth' },
  { label: t('audit.logList.filterOptions.role'), value: 'role' },
  { label: t('audit.logList.filterOptions.permission'), value: 'permission' },
  { label: t('audit.logList.filterOptions.session'), value: 'session' },
]);

const resultOptions = computed(() => [
  { label: t('audit.logList.filterOptions.allResults'), value: 'all' },
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'SUCCESS' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'FAILED' },
  { label: t('audit.logList.filterOptions.DENIED'), value: 'DENIED' },
  { label: t('audit.logList.filterOptions.ERROR'), value: 'ERROR' },
]);

const riskOptions = computed(() => [
  { label: t('audit.logList.filterOptions.allRisk'), value: 'all' },
  { label: t('audit.logList.filterOptions.LOW'), value: 'LOW' },
  { label: t('audit.logList.filterOptions.MEDIUM'), value: 'MEDIUM' },
  { label: t('audit.logList.filterOptions.HIGH'), value: 'HIGH' },
  { label: t('audit.logList.filterOptions.CRITICAL'), value: 'CRITICAL' },
]);

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
</script>
<style scoped lang="less">
.audit-filters {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.audit-filters__row {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.audit-filters__input,
.audit-filters__keyword,
.audit-filters__date {
  min-width: 0;
}

@media (width <= 1280px) {
  .audit-filters__row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .audit-filters__row {
    grid-template-columns: 1fr;
  }
}
</style>
