<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="announcement-user-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="announcement.user.title"
        description-key="announcement.user.description"
        :source="{ labelKey: 'announcement.route.user.title', fallback: t('announcement.route.user.title') }"
      >
        <template #actions>
          <t-checkbox v-model="filters.unreadOnly" class="announcement-user-page__unread-filter">
            {{ t('announcement.user.unreadOnly') }}
          </t-checkbox>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchAnnouncements">
            {{ t('announcement.user.refresh') }}
          </t-button>
          <t-button theme="primary" :disabled="!canMarkAllRead" :loading="markingAllRead" @click="markAllRead">
            {{ t('announcement.user.markAllRead') }}
          </t-button>
        </template>
      </management-page-header>

      <t-card class="announcement-user-page__surface" size="small" :bordered="true">
        <template #header>
          <div class="announcement-user-page__surface-head">
            <div>
              <strong>{{ t('announcement.user.summary', { count: total }) }}</strong>
              <span>{{ t('announcement.user.listHint') }}</span>
            </div>
            <t-tag :theme="unreadCount > 0 ? 'primary' : 'default'" variant="light">
              {{ t('announcement.user.unreadSummary', { count: unreadCount }) }}
            </t-tag>
          </div>
        </template>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('announcement.user.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchAnnouncements">
              {{ t('announcement.user.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-loading v-else :loading="loading" size="large" :text="t('announcement.user.loading')">
          <t-list v-if="presentedRows.length" class="announcement-user-page__list" :split="true" size="large">
            <t-list-item v-for="row in presentedRows" :key="row.id">
              <article
                class="announcement-user-page__item"
                :class="{ 'is-unread': row.unread }"
                role="button"
                tabindex="0"
                @click="openReadPanel(row)"
                @keydown.enter.prevent="openReadPanel(row)"
                @keydown.space.prevent="openReadPanel(row)"
              >
                <span v-if="row.unread" class="announcement-user-page__unread-dot" aria-hidden="true" />
                <div class="announcement-user-page__item-main">
                  <header class="announcement-user-page__item-head">
                    <div class="announcement-user-page__title-group">
                      <strong>{{ row.title }}</strong>
                      <div class="announcement-user-page__tags">
                        <t-tag v-if="row.pinned" theme="primary" variant="light" size="small">
                          {{ row.pinnedLabel }}
                        </t-tag>
                        <t-tag :theme="row.levelTheme" variant="light" size="small">
                          {{ row.levelLabel }}
                        </t-tag>
                        <t-tag :theme="row.unread ? 'primary' : 'default'" variant="light" size="small">
                          {{ row.unreadLabel }}
                        </t-tag>
                      </div>
                    </div>
                    <t-button
                      v-if="row.unread"
                      theme="primary"
                      variant="text"
                      size="small"
                      :loading="markingReadId === row.id"
                      @click.stop="markRead(row.id)"
                    >
                      {{ t('announcement.user.markRead') }}
                    </t-button>
                  </header>
                  <t-tooltip placement="top-left" :content="row.summary">
                    <p class="announcement-user-page__summary">{{ row.summary }}</p>
                  </t-tooltip>
                  <dl class="announcement-user-page__meta">
                    <div>
                      <dt>{{ t('announcement.user.publishAt') }}</dt>
                      <dd>{{ row.publishAtLabel }}</dd>
                    </div>
                    <div>
                      <dt>{{ t('announcement.user.expireAt') }}</dt>
                      <dd>{{ row.expireAtLabel }}</dd>
                    </div>
                    <div>
                      <dt>{{ t('announcement.user.readAt') }}</dt>
                      <dd>{{ row.readAtLabel }}</dd>
                    </div>
                  </dl>
                </div>
              </article>
            </t-list-item>
          </t-list>

          <t-empty v-else class="announcement-user-page__empty" :title="emptyTitle" :description="emptyDescription">
            <template v-if="filters.unreadOnly" #action>
              <t-button theme="default" variant="outline" @click="filters.unreadOnly = false">
                {{ t('announcement.user.showAll') }}
              </t-button>
            </template>
          </t-empty>
        </t-loading>

        <template #footer>
          <management-table-pagination :summary="t('announcement.user.footerTotal', { count: total })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :page-size-options="[10, 20, 50]"
              :show-page-number="true"
              @change="handlePageChange"
            />
          </management-table-pagination>
        </template>
      </t-card>
    </management-page-content>

    <announcement-read-panel
      :visible="readPanelVisible"
      :announcement="readPanelRecord"
      source="center"
      :marking-read="markingReadId === readPanelRecord?.id"
      @close="closeReadPanel"
      @mark-read="markReadFromPanel"
    />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTablePagination,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  getAnnouncementUnreadCount,
  getMyAnnouncements,
  markAllAnnouncementsRead,
  markAnnouncementRead,
} from '../../api/announcement';
import AnnouncementReadPanel from '../../components/AnnouncementReadPanel.vue';
import { emitAnnouncementChanged, onAnnouncementChanged } from '../../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../../domain/announcement-presenter';

const { locale, t } = useI18n();

const loading = ref(false);
const markingAllRead = ref(false);
const markingReadId = ref<number | null>(null);
const listError = ref('');
const rows = ref<AnnouncementViewModel[]>([]);
const readPanelRecord = ref<AnnouncementViewModel | null>(null);
const readPanelVisible = ref(false);
const total = ref(0);
const unreadCount = ref(0);
const pagination = reactive({
  current: 1,
  pageSize: 20,
});
const filters = reactive({
  unreadOnly: false,
});
let stopAnnouncementChanged: (() => void) | undefined;
let suppressNextChangedRefresh = false;

const presentedRows = computed(() => rows.value);
const canMarkAllRead = computed(() => unreadCount.value > 0 && !markingAllRead.value);
const emptyTitle = computed(() =>
  filters.unreadOnly ? t('announcement.user.emptyUnreadTitle') : t('announcement.user.emptyTitle'),
);
const emptyDescription = computed(() =>
  filters.unreadOnly ? t('announcement.user.emptyUnreadDescription') : t('announcement.user.emptyDescription'),
);

onMounted(() => {
  void fetchAnnouncements();
  stopAnnouncementChanged = onAnnouncementChanged(handleAnnouncementChanged);
});

onBeforeUnmount(() => {
  stopAnnouncementChanged?.();
});

watch(() => filters.unreadOnly, handleUnreadOnlyChange);
watch(() => `${pagination.current}:${pagination.pageSize}`, fetchAnnouncements);

async function fetchAnnouncements() {
  loading.value = true;
  listError.value = '';
  try {
    const [page, count] = await Promise.all([
      getMyAnnouncements({
        page: pagination.current,
        page_size: pagination.pageSize,
        unread_only: filters.unreadOnly || undefined,
      }),
      getAnnouncementUnreadCount(),
    ]);
    rows.value = page.items.map((item) => presentAnnouncement(item, t, locale.value));
    total.value = page.total;
    unreadCount.value = count.count;
    pagination.current = page.page || pagination.current;
    pagination.pageSize = page.page_size || pagination.pageSize;
  } catch (error) {
    listError.value = resolveLocalizedErrorMessage(t, error, t('announcement.user.loadFailed'));
    rows.value = [];
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

async function markRead(id: number) {
  markingReadId.value = id;
  try {
    const updated = await markAnnouncementRead(id);
    const updatedView = presentAnnouncement(updated, t, locale.value);
    readPanelRecord.value = readPanelRecord.value?.id === id ? updatedView : readPanelRecord.value;
    await fetchAnnouncements();
    emitLocalAnnouncementChanged();
    MessagePlugin.success(t('announcement.user.markReadSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('announcement.user.markReadFailed')));
  } finally {
    markingReadId.value = null;
  }
}

function openReadPanel(row: AnnouncementViewModel) {
  readPanelRecord.value = row;
  readPanelVisible.value = true;
}

function closeReadPanel() {
  readPanelVisible.value = false;
}

async function markReadFromPanel() {
  if (!readPanelRecord.value) {
    return;
  }

  await markRead(readPanelRecord.value.id);
  if (!readPanelRecord.value?.unread) {
    readPanelVisible.value = false;
  }
}

async function markAllRead() {
  markingAllRead.value = true;
  try {
    await markAllAnnouncementsRead();
    await fetchAnnouncements();
    emitLocalAnnouncementChanged();
    MessagePlugin.success(t('announcement.user.markAllReadSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('announcement.user.markAllReadFailed')));
  } finally {
    markingAllRead.value = false;
  }
}

function handlePageChange() {
  void fetchAnnouncements();
}

function handleUnreadOnlyChange() {
  pagination.current = 1;
  void fetchAnnouncements();
}

function handleAnnouncementChanged() {
  if (suppressNextChangedRefresh) {
    return;
  }
  void fetchAnnouncements();
}

function emitLocalAnnouncementChanged() {
  suppressNextChangedRefresh = true;
  emitAnnouncementChanged();
  queueMicrotask(() => {
    suppressNextChangedRefresh = false;
  });
}
</script>
<style scoped lang="less">
.announcement-user-page {
  min-width: 0;
}

.announcement-user-page__unread-filter {
  margin-right: var(--graft-density-gap-4);
}

.announcement-user-page__surface {
  :deep(.t-card__header) {
    border-bottom: 1px solid var(--td-component-stroke);
    padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl);
  }

  :deep(.t-card__body) {
    padding: 0;
  }
}

.announcement-user-page__surface-head {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.announcement-user-page__surface-head > div {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.announcement-user-page__surface-head strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.announcement-user-page__surface-head span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.announcement-user-page__list {
  min-height: 240px;
}

.announcement-user-page__item {
  align-items: flex-start;
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-10);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl);
  width: 100%;
}

.announcement-user-page__item:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: -2px;
}

.announcement-user-page__item.is-unread {
  background: var(--td-brand-color-light);
}

.announcement-user-page__unread-dot {
  background: var(--td-brand-color);
  border-radius: 50%;
  flex: 0 0 auto;
  height: 8px;
  margin-top: var(--graft-density-gap-8);
  width: 8px;
}

.announcement-user-page__item-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.announcement-user-page__item-head {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.announcement-user-page__title-group {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.announcement-user-page__title-group strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  overflow-wrap: anywhere;
}

.announcement-user-page__tags {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
}

.announcement-user-page__summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.announcement-user-page__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12) var(--graft-density-gap-20);
  margin: 0;
}

.announcement-user-page__meta div {
  display: flex;
  gap: var(--graft-density-gap-6);
}

.announcement-user-page__meta dt {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.announcement-user-page__meta dd {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.announcement-user-page__empty {
  padding: var(--td-comp-paddingTB-xxl) var(--td-comp-paddingLR-xl);
}

@media (width <= 768px) {
  .announcement-user-page__surface-head,
  .announcement-user-page__item-head {
    align-items: stretch;
    flex-direction: column;
  }

  .announcement-user-page__item {
    padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
  }

  .announcement-user-page__meta,
  .announcement-user-page__meta div {
    flex-direction: column;
    gap: var(--graft-density-gap-4);
  }
}
</style>
