<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <announcement-read-panel
    :visible="visible"
    :announcement="current"
    source="popup"
    :marking-read="markingRead"
    @close="dismissCurrent"
    @mark-read="markCurrentRead"
    @open-center="openCenter"
  />
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { markAnnouncementRead } from '../api/announcement';
import { ANNOUNCEMENT_ROUTE_PATH } from '../contract/paths';
import { emitAnnouncementChanged, onAnnouncementChanged } from '../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../domain/announcement-presenter';
import { loadUnreadAnnouncementCandidate } from './announcement-read-panel';
import AnnouncementReadPanel from './AnnouncementReadPanel.vue';

const { locale: activeLocale, t: translate } = useI18n();
const announcementRouter = useRouter();

const visible = ref(false);
const markingRead = ref(false);
const currentItem = ref<AnnouncementViewModel | null>(null);
const dismissedIds = new Set<number>();
let stopAnnouncementChanged: (() => void) | undefined;

const current = computed(() => currentItem.value);

onMounted(() => {
  void refreshPopupCandidate();
  stopAnnouncementChanged = onAnnouncementChanged(refreshPopupCandidate);
});

onBeforeUnmount(() => {
  stopAnnouncementChanged?.();
});

async function refreshPopupCandidate() {
  if (visible.value) {
    return;
  }

  try {
    currentItem.value = await loadUnreadAnnouncementCandidate({
      filter: (item) =>
        item.delivery_mode === 'popup' && !item.read_at && item.unread !== false && !dismissedIds.has(item.id),
      locale: activeLocale.value,
      pageSize: 10,
      t: translate,
    });
    visible.value = Boolean(currentItem.value);
  } catch {
    currentItem.value = null;
    visible.value = false;
  }
}

function dismissCurrent() {
  if (currentItem.value) {
    dismissedIds.add(currentItem.value.id);
  }
  visible.value = false;
}

async function markCurrentRead() {
  if (!currentItem.value) {
    return;
  }

  markingRead.value = true;
  try {
    const updated = await markAnnouncementRead(currentItem.value.id);
    dismissedIds.add(currentItem.value.id);
    currentItem.value = presentAnnouncement(updated, translate, activeLocale.value);
    visible.value = false;
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(translate, error, translate('announcement.popup.markReadFailed')));
  } finally {
    markingRead.value = false;
  }
}

function openCenter() {
  dismissCurrent();
  void announcementRouter.push(ANNOUNCEMENT_ROUTE_PATH.USER_LIST);
}
</script>
