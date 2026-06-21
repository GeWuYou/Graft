<script setup lang="ts">
const condition = true;
const userAction = {
  key: 'toggleUser',
  fallbackLabel: condition ? '启用用户' : '禁用用户',
};
</script>

<template>
  <button>{{ userAction.key }}</button>
</template>
