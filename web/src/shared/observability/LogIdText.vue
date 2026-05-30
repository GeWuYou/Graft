<template>
  <span class="log-id-text" :title="tooltip ?? undefined">
    {{ displayValue }}
  </span>
</template>
<script setup lang="ts">
const props = defineProps<{
  displayValue?: string | null;
  tooltip?: string | null;
}>();

void props;
</script>
<style scoped lang="less">
.log-id-text {
  display: inline-block;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 600;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}
</style>
