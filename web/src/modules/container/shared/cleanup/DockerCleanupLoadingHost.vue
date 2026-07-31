<template>
  <t-loading :loading="loading">
    <div class="docker-cleanup-loading-host">
      <slot />
    </div>
  </t-loading>
</template>
<script setup lang="ts">
/**
 * 为 Docker 资源清理对话框提供稳定的加载覆盖区域，避免候选结果尚未返回时遮罩宿主高度塌陷。
 */
defineOptions({ name: 'DockerCleanupLoadingHost' });

defineProps<{ loading: boolean }>();
</script>
<style scoped>
.docker-cleanup-loading-host {
  min-block-size: 10rem;
}
</style>
