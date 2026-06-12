<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-dialog
    v-model:visible="visible"
    class="announcement-popup-dialog"
    placement="center"
    width="560px"
    :close-on-overlay-click="false"
    :confirm-btn="null"
    :cancel-btn="null"
    :footer="false"
    destroy-on-close
    @close="dismissCurrent"
  >
    <template #header>
      <div class="announcement-popup-dialog__header">
        <t-tag v-if="current" :theme="current.levelTheme" variant="light">
          {{ current.levelLabel }}
        </t-tag>
        <t-tag theme="primary" variant="light">
          {{ t('announcement.readState.unread') }}
        </t-tag>
      </div>
    </template>

    <article v-if="current" class="announcement-popup-dialog__body">
      <h2>{{ current.title }}</h2>
      <p>{{ current.publishAtLabel }}</p>
      <safe-markdown class="announcement-popup-dialog__content" :source="current.content" />
    </article>

    <template #footer>
      <div class="announcement-popup-dialog__footer">
        <t-button theme="default" variant="outline" @click="dismissCurrent">
          {{ t('announcement.popup.viewLater') }}
        </t-button>
        <t-button theme="primary" :loading="markingRead" @click="markCurrentRead">
          {{ t('announcement.popup.markRead') }}
        </t-button>
      </div>
    </template>
  </t-dialog>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { SafeMarkdown } from '@/shared/components/markdown';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getMyAnnouncements, markAnnouncementRead } from '../api/announcement';
import { emitAnnouncementChanged, onAnnouncementChanged } from '../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../domain/announcement-presenter';

const { locale, t } = useI18n();

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
    const page = await getMyAnnouncements({
      page: 1,
      page_size: 10,
      unread_only: true,
    });
    const popup = page.items.find((item) => item.delivery_mode === 'popup' && !dismissedIds.has(item.id));
    currentItem.value = popup ? presentAnnouncement(popup, t, locale.value) : null;
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
    await markAnnouncementRead(currentItem.value.id);
    dismissedIds.add(currentItem.value.id);
    visible.value = false;
    currentItem.value = null;
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('announcement.popup.markReadFailed')));
  } finally {
    markingRead.value = false;
  }
}
</script>
<style scoped lang="less">
.announcement-popup-dialog__header,
.announcement-popup-dialog__footer {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
}

.announcement-popup-dialog__body {
  align-items: stretch;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  text-align: left;
}

.announcement-popup-dialog__body h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
  overflow-wrap: anywhere;
}

.announcement-popup-dialog__body > p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.announcement-popup-dialog__content {
  border-top: 1px solid var(--td-component-stroke);
  margin-top: var(--graft-density-gap-6);
  max-height: min(52vh, 420px);
  overflow: auto;
  padding-top: var(--graft-density-gap-12);
  width: 100%;
}

.announcement-popup-dialog__footer {
  justify-content: flex-end;
}
</style>
