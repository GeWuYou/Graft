<template>
  <t-space size="small" break-line>
    <t-button theme="primary" variant="outline" @click="$emit('import')">
      {{ importLabel }}
    </t-button>
    <t-button theme="default" variant="outline" @click="$emit('create')">
      {{ createLabel }}
    </t-button>
    <t-button v-if="showReset" theme="default" variant="outline" @click="$emit('reset')">
      {{ resetLabel }}
    </t-button>
  </t-space>
</template>
<script setup lang="ts">
defineProps<{
  importLabel: string;
  createLabel: string;
  resetLabel: string;
  showReset?: boolean;
}>();

defineEmits<{
  import: [];
  create: [];
  reset: [];
}>();
</script>
