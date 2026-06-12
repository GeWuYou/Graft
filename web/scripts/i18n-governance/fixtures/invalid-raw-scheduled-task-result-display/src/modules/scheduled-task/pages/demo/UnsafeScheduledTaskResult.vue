<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section>
    <p>{{ row.result_summary }}</p>
    <p>{{ row.error_message }}</p>
    <t-alert :message="actionResultStructured.summary" />
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
};

function unsafeMultilineReturn() {
  return row.error_message;
}

function emitResult(value: string) {
  return value;
}
</script>
