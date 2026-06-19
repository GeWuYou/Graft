<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <span class="announcement-header-entry">
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

    <announcement-read-panel
      :visible="readPanelVisible"
      :announcement="readPanelRecord"
      source="header"
      :marking-read="markingRead"
      @close="closeReadPanel"
      @mark-read="markCurrentRead"
      @open-center="openCenter"
    />
  </span>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getAnnouncementUnreadCount, markAnnouncementRead } from '../api/announcement';
import { ANNOUNCEMENT_ROUTE_PATH } from '../contract/paths';
import { emitAnnouncementChanged, onAnnouncementChanged } from '../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../domain/announcement-presenter';
import { loadUnreadAnnouncementCandidate } from './announcement-read-panel';
import AnnouncementReadPanel from './AnnouncementReadPanel.vue';

const { locale, t } = useI18n();
const router = useRouter();

const loading = ref(false);
const markingRead = ref(false);
const unreadCount = ref(0);
const readPanelRecord = ref<AnnouncementViewModel | null>(null);
const readPanelVisible = ref(false);
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

async function openAnnouncements() {
  loading.value = true;
  try {
    const latestUnread = await loadUnreadAnnouncementCandidate({
      locale: locale.value,
      pageSize: 1,
      t,
    });
    if (latestUnread) {
      readPanelRecord.value = latestUnread;
      readPanelVisible.value = true;
      return;
    }

    openCenter();
  } catch {
    openCenter();
  } finally {
    loading.value = false;
  }
}

function closeReadPanel() {
  readPanelVisible.value = false;
}

function openCenter() {
  readPanelVisible.value = false;
  void router.push(ANNOUNCEMENT_ROUTE_PATH.USER_LIST);
}

async function markCurrentRead() {
  if (!readPanelRecord.value) {
    return;
  }

  markingRead.value = true;
  try {
    const updated = await markAnnouncementRead(readPanelRecord.value.id);
    readPanelRecord.value = presentAnnouncement(updated, t, locale.value);
    readPanelVisible.value = false;
    emitAnnouncementChanged();
    await refreshUnreadCount();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('announcement.header.markReadFailed')));
  } finally {
    markingRead.value = false;
  }
}
</script>
<style scoped lang="less">
.announcement-header-entry {
  align-items: center;
  display: inline-flex;
  height: var(--td-comp-size-m);
  justify-content: center;
  width: var(--td-comp-size-m);
}
</style>
