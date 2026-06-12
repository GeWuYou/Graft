<template>
  <section>
    <p>{{ row.result_summary }}</p>
    <p>{{ row.error_message }}</p>
    <t-alert :message="actionResultStructured.summary" />
  </section>
</template>

<script setup lang="ts">
const row = {
  result_summary: 'deleted 3 rows',
  error_message: 'retention window is invalid',
};

const actionResultStructured = {
  summary: 'deleted 3 rows',
};
</script>
