<template>
  <t-space size="small" break-line>
    <t-button theme="primary" @click="$emit('create')">
      {{ createLabel }}
    </t-button>
    <t-button v-if="showReset" theme="default" variant="outline" @click="$emit('reset')">
      {{ resetLabel }}
    </t-button>
  </t-space>
</template>
<script setup lang="ts">
defineProps<{
  createLabel: string;
  resetLabel: string;
  showReset?: boolean;
}>();

defineEmits<{
  create: [];
  reset: [];
}>();
</script>
