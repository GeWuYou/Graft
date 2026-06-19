<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="header-notice-actions">
    <component :is="announcementHeaderEntry" />
    <component :is="notificationHeaderWidget" />
    <component :is="announcementPopupHost" />
  </div>
</template>
<script setup lang="ts">
import { announcementHeaderEntry, announcementPopupHost } from '@/modules/announcement';
import { notificationHeaderWidget } from '@/modules/notification';
</script>
<style scoped lang="less">
.header-notice-actions {
  align-items: center;
  display: inline-flex;
  flex: 0 0 auto;
  gap: 0;
  height: var(--td-comp-size-m);
  justify-content: center;

  :deep(.announcement-header-entry),
  :deep(.notification-header-entry) {
    flex: 0 0 var(--td-comp-size-m);
  }
}
</style>
