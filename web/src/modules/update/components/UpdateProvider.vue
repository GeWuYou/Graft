<template><span class="update-provider" aria-hidden="true" /></template>
<script setup lang="ts">
// Provider 只在认证后的后台壳挂载一次，集中触发已授权的发现快照加载并在会话结束时失效。
import { onBeforeUnmount, onMounted } from 'vue';

import { useUpdateDiscoveryStore } from '../store/discovery';

const discoveryStore = useUpdateDiscoveryStore();

onMounted(() => {
  void discoveryStore.ensureSnapshot();
});

onBeforeUnmount(() => {
  discoveryStore.reset();
});
</script>
<style scoped lang="less">
.update-provider {
  display: none;
}
</style>
