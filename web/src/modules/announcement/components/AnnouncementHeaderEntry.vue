<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-tooltip placement="bottom" :content="t('announcement.header.title')">
    <t-badge :count="unreadCount" :max-count="99" :offset="[4, 4]">
      <t-button
        theme="default"
        shape="square"
        variant="text"
        :loading="loading"
        :aria-label="t('announcement.header.title')"
        :title="t('announcement.header.title')"
        @click="openAnnouncements"
      >
        <t-icon name="notification" />
      </t-button>
    </t-badge>
  </t-tooltip>
</template>
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getAnnouncementUnreadCount } from '../api/announcement';
import { ANNOUNCEMENT_ROUTE_PATH } from '../contract/paths';
import { onAnnouncementChanged } from '../contract/refresh';

const { t } = useI18n();
const router = useRouter();

const loading = ref(false);
const unreadCount = ref(0);
let stopAnnouncementChanged: (() => void) | undefined;

onMounted(() => {
  void refreshUnreadCount();
  stopAnnouncementChanged = onAnnouncementChanged(refreshUnreadCount);
});

onBeforeUnmount(() => {
  stopAnnouncementChanged?.();
});

async function refreshUnreadCount() {
  loading.value = true;
  try {
    const response = await getAnnouncementUnreadCount();
    unreadCount.value = response.count;
  } catch {
    unreadCount.value = 0;
  } finally {
    loading.value = false;
  }
}

function openAnnouncements() {
  void router.push(ANNOUNCEMENT_ROUTE_PATH.USER_LIST);
}
</script>
