<template>
  <ActionMenu :items="items" more-label-fallback="更多" />
</template>

<script setup lang="ts">
const items = [{ key: 'refresh', labelKey: 'demo.actions.refresh' }];
</script>
