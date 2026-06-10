<!-- Copyright (c) 2025-2026 GeWuYou -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script setup lang="ts">
const enabled = true;
const count = 3;
</script>

<template>
  <t-button :fallback-label="enabled ? '启用用户' : '禁用用户'" />
  <t-empty v-bind:empty-description="count > 0 ? `还有 ${count} 条任务` : '没有任务'" />
</template>
