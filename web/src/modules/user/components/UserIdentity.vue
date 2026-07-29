<template>
  <div class="user-cell__avatar">{{ initial }}</div>
  <div class="user-cell__meta">
    <span class="user-cell__display">{{ displayName }}</span>
    <span class="user-cell__username">{{ user.email || `@${user.username}` }}</span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

// 用户身份摘要在卡片和表格中共用，避免两种布局漂移为不同展示规则。
const props = defineProps<{
  user: {
    display?: string;
    email?: string | null;
    username: string;
  };
}>();
const displayName = computed(() => props.user.display?.trim() || props.user.username);
const initial = computed(() => Array.from(displayName.value)[0]?.toUpperCase() || '?');
</script>
