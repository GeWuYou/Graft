<template>
  <t-loading :loading="loading">
    <div class="docker-cleanup-loading-host" :class="{ 'docker-cleanup-loading-host--empty': empty }">
      <slot />
    </div>
  </t-loading>
</template>
<script setup lang="ts">
/**
 * 为 Docker 资源清理对话框提供稳定的加载覆盖区域，并在候选为空时居中展示空态，避免内容区高度塌陷或页面间布局漂移。
 */
defineOptions({ name: 'DockerCleanupLoadingHost' });

withDefaults(defineProps<{ empty?: boolean; loading: boolean }>(), { empty: false });
</script>
<style scoped>
.docker-cleanup-loading-host {
  min-block-size: 10rem;
}

.docker-cleanup-loading-host--empty {
  display: grid;
  place-items: center;
}
</style>
