<template>
  <section>
    <strong>{{ item.event_type }}</strong>
    <span>{{ item.resource_type }}</span>
    <span>{{ item.target_type }}</span>
    <p>{{ fallbackTitle }}</p>
  </section>
</template>
<script setup lang="ts">
const item = {
  event_type: 'task_succeeded',
  resource_type: 'scheduled_task_run',
  target_type: 'USER',
};
const fallbackTitle = 'Scheduled task succeeded';
</script>
