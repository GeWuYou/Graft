<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section>
    <p>{{ runResultText(row) }}</p>
    <t-alert :message="localizedStructuredRunResultText(actionResultStructured)" />
    <button @click="emitResult(row.result_summary)">{{ t('scheduledTask.list.result.emit') }}</button>
  </section>
</template>

<script setup lang="ts">
function t(key: string) {
  return key;
}

const row = {
  result_summary: 'deleted 3 rows',
  error_message: 'retention window is invalid',
};

const actionResultStructured = {
  summary: 'deleted 3 rows',
  metrics: {
    deletedCount: 3,
  },
};

function runResultText(run: typeof row) {
  return run ? t('scheduledTask.list.result.completed') : t('scheduledTask.list.detail.none');
}

function localizedStructuredRunResultText(result: typeof actionResultStructured) {
  return result.metrics.deletedCount ? t('scheduledTask.list.result.deletedRows') : t('scheduledTask.list.detail.none');
}
</script>
