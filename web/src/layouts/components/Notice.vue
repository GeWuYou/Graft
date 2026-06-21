<template>
  <div class="header-operation-item">
    <component :is="notificationHeaderWidget" />
  </div>
  <div class="header-operation-item">
    <component :is="announcementHeaderEntry" />
  </div>
  <component :is="announcementPopupHost" />
</template>
<script setup lang="ts">
import { announcementHeaderEntry, announcementPopupHost } from '@/modules/announcement';
import { notificationHeaderWidget } from '@/modules/notification';
</script>
