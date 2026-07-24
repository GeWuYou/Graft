<template>
  <update-center :data-source="dataSource" />
</template>
<script setup lang="ts">
// 开发专用路由复用正式页面，仅以内存数据源隔离真实更新 API 和部署环境。
import { createUpdateCenterPreviewDataSource } from '../../mock/update-center';
import UpdateCenter from '../center/index.vue';

const dataSource = createUpdateCenterPreviewDataSource();
</script>
