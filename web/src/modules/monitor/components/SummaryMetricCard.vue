<template>
  <governance-summary-card
    class="server-status-summary-card"
    kind="status"
    :title="title"
    :value="value"
    :value-aside="valueAside"
    :description="description"
  >
    <template v-if="statusLabel" #badge>
      <status-tag :label="statusLabel" :status="status" />
    </template>
  </governance-summary-card>
</template>
<script setup lang="ts">
import { GovernanceSummaryCard } from '@/shared/components/governance';

import type { ServerStatusTone } from './server-status-ui';
import StatusTag from './StatusTag.vue';

withDefaults(
  defineProps<{
    title: string;
    value: string;
    description: string;
    status?: ServerStatusTone;
    statusLabel?: string;
    valueAside?: string;
  }>(),
  {
    status: 'unknown',
    statusLabel: '',
    valueAside: '',
  },
);
</script>
