<template>
  <safe-markdown :source="source" />
</template>
<script setup lang="ts">
import SafeMarkdown from './SafeMarkdown.vue';

defineProps<{
  source?: string | null;
}>();
</script>
