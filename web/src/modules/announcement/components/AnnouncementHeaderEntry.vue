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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { ANNOUNCEMENT_ROUTE_PATH } from '../contract/paths';
import { emitAnnouncementChanged, onAnnouncementChanged } from '../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../domain/announcement-presenter';
import {
  invalidateMyAnnouncementQueries,
  useAnnouncementUnreadCountQuery,
  useMarkAnnouncementReadMutation,
} from '../shared/announcement-queries';
import { loadUnreadAnnouncementCandidate } from './announcement-read-panel';
import AnnouncementReadPanel from './AnnouncementReadPanel.vue';

const { locale, t } = useI18n();
const router = useRouter();

const openingAnnouncements = ref(false);
const readPanelRecord = ref<AnnouncementViewModel | null>(null);
const readPanelVisible = ref(false);
const unreadCountQuery = useAnnouncementUnreadCountQuery();
const markAnnouncementReadMutation = useMarkAnnouncementReadMutation();
const loading = computed(() => openingAnnouncements.value || unreadCountQuery.isFetching.value);
const markingRead = computed(() => markAnnouncementReadMutation.isPending.value);
const unreadCount = computed(() => unreadCountQuery.data.value?.count ?? 0);
let stopAnnouncementChanged: (() => void) | undefined;

onMounted(() => {
  stopAnnouncementChanged = onAnnouncementChanged(handleAnnouncementChanged);
});

onBeforeUnmount(() => {
  stopAnnouncementChanged?.();
});

function handleAnnouncementChanged() {
  void invalidateMyAnnouncementQueries();
}

async function openAnnouncements() {
  openingAnnouncements.value = true;
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
    openingAnnouncements.value = false;
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

  try {
    const updated = await markAnnouncementReadMutation.mutateAsync(readPanelRecord.value.id);
    readPanelRecord.value = presentAnnouncement(updated, t, locale.value);
    readPanelVisible.value = false;
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('announcement.header.markReadFailed')));
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
