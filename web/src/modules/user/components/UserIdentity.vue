<template>
  <div class="user-cell__avatar">{{ initial }}</div>
  <div class="user-cell__meta">
    <span class="user-cell__display">{{ user.display || user.username }}</span>
    <span class="user-cell__username">{{ user.email || `@${user.username}` }}</span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

// 用户身份摘要在卡片和表格中共用，避免两种布局漂移为不同展示规则。
const props = defineProps<{
  user: {
    display?: string;
    email?: string;
    username: string;
  };
}>();
const initial = computed(() => (props.user.display || props.user.username).trim().slice(0, 1).toUpperCase() || '?');
</script>
