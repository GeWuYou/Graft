<template>
  <server-status-page-shell
    :eyebrow="eyebrow"
    :title="title"
    :description="description"
    :compact-header="compactHeader"
  >
    <template #toolbar>
      <monitor-page-toolbar
        :auto-refresh-enabled="autoRefreshEnabled"
        :loading="loading"
        :pause-auto-refresh-label="pauseAutoRefreshLabel"
        :refresh-interval-label="refreshIntervalLabel"
        :refresh-interval-options="refreshIntervalOptions"
        :refresh-interval-value="refreshIntervalValue"
        :refresh-now-label="refreshNowLabel"
        :resume-auto-refresh-label="resumeAutoRefreshLabel"
        :status="status"
        :status-label="statusLabel"
        :trend-range-label-placeholder="trendRangeLabelPlaceholder"
        @refresh="$emit('refresh')"
        @toggle-auto-refresh="$emit('toggle-auto-refresh')"
        @update:refresh-interval-value="$emit('update:refresh-interval-value', $event)"
      />
    </template>

    <template v-if="$slots.headerHint" #headerHint>
      <slot name="headerHint" />
    </template>

    <template #summary>
      <monitor-page-summary :items="summaryItems" />
    </template>

    <template #feedback>
      <monitor-page-feedback :title="errorTitle" :message="errorMessage" />
    </template>

    <slot />

    <t-empty v-if="initialized && !hasServerStatus && !loading" :description="emptyDescription" />
  </server-status-page-shell>
</template>
<script setup lang="ts">
import type { MonitorStatusPageFrameProps } from '../shared/frame-props';
import MonitorPageFeedback from './MonitorPageFeedback.vue';
import MonitorPageSummary from './MonitorPageSummary.vue';
import MonitorPageToolbar from './MonitorPageToolbar.vue';
import ServerStatusPageShell from './ServerStatusPageShell.vue';

defineProps<MonitorStatusPageFrameProps>();

defineEmits<{
  refresh: [];
  'toggle-auto-refresh': [];
  'update:refresh-interval-value': [value: number | string];
}>();
</script>
