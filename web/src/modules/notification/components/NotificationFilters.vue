<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="notification-filters">
    <t-select
      v-model="filterModel.severity"
      clearable
      class="notification-filters__select"
      :options="severityOptions"
      :placeholder="t('notification.filters.severity')"
    />
    <t-select
      v-model="filterModel.category"
      clearable
      class="notification-filters__select"
      :options="categoryOptions"
      :placeholder="t('notification.filters.category')"
    />
    <t-select
      v-model="filterModel.sourceModule"
      clearable
      filterable
      class="notification-filters__select"
      :options="sourceOptions"
      :placeholder="t('notification.filters.sourceModule')"
    />
    <t-date-range-picker
      v-model="filterModel.occurredRange"
      clearable
      enable-time-picker
      class="notification-filters__range"
      format="YYYY-MM-DD HH:mm:ss"
      value-type="YYYY-MM-DD HH:mm:ss"
      :placeholder="[t('notification.filters.occurredFrom'), t('notification.filters.occurredTo')]"
    />
    <t-button theme="primary" :loading="loading" @click="$emit('search')">
      {{ t('notification.actions.search') }}
    </t-button>
    <t-button theme="default" variant="text" @click="$emit('reset')">
      {{ t('notification.actions.reset') }}
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { NOTIFICATION_CATEGORY_VALUES } from '../contract/category';
import { NOTIFICATION_SEVERITY_VALUES } from '../contract/severity';
import { notificationSourceLabel } from '../shared/presentation';
import type { NotificationFilterState } from '../types/notification';

const props = defineProps<{
  loading?: boolean;
  sourceModules: string[];
}>();

defineEmits<{
  (e: 'reset'): void;
  (e: 'search'): void;
}>();

const filterModel = defineModel<NotificationFilterState>({ required: true });
const { t } = useI18n();

const severityOptions = computed(() =>
  NOTIFICATION_SEVERITY_VALUES.map((value) => ({
    label: t(`notification.severity.${value}`),
    value,
  })),
);
const categoryOptions = computed(() =>
  NOTIFICATION_CATEGORY_VALUES.map((value) => ({
    label: t(`notification.category.${value}`),
    value,
  })),
);
const sourceOptions = computed(() =>
  props.sourceModules.map((sourceModule) => ({
    label: notificationSourceLabel(sourceModule, t),
    value: sourceModule,
  })),
);
</script>
<style scoped lang="less">
.notification-filters {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.notification-filters__select {
  width: 180px;
}

.notification-filters__range {
  width: 360px;
}
</style>
