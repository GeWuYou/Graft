<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <safe-markdown :source="source" />
</template>
<script setup lang="ts">
import SafeMarkdown from './SafeMarkdown.vue';

defineProps<{
  source?: string | null;
}>();
</script>
