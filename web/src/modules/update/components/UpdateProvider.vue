<template><span class="update-provider" aria-hidden="true" /></template>
<script setup lang="ts">
// Provider 只在认证后的后台壳挂载一次，集中触发已授权的发现快照加载并在会话结束时失效。
import { onBeforeUnmount, onMounted } from 'vue';

import { useUpdateDiscoveryStore } from '../store/discovery';
import { useUpdateProgressStore } from '../store/progress';

const discoveryStore = useUpdateDiscoveryStore();
const progressStore = useUpdateProgressStore();

onMounted(() => {
  void discoveryStore.ensureSnapshot();
  void progressStore.resume();
  document.addEventListener('visibilitychange', handleVisibilityChange, false);
});

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  discoveryStore.reset();
  progressStore.reset();
});

function handleVisibilityChange() {
  // 定时检查由服务端调度；前台恢复时只读取该快照，绝不触发 GitHub 检查请求。
  if (document.visibilityState === 'visible') {
    void discoveryStore.revalidateVisibleSnapshot();
  }
}
</script>
<style scoped lang="less">
.update-provider {
  display: none;
}
</style>
