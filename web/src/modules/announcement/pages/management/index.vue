<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="announcement-management-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="announcement.management.title"
        description-key="announcement.management.description"
        :source="{ labelKey: 'menu.server.title', fallback: t('menu.server.title') }"
      >
        <template #actions>
          <div class="announcement-management-page__header-actions">
            <t-button theme="default" variant="outline" :loading="loading" @click="fetchAnnouncements">
              {{ t('announcement.management.refresh') }}
            </t-button>
            <t-button
              v-permission="permissionCodes.CREATE"
              theme="primary"
              data-testid="announcement-create"
              @click="openCreateDrawer"
            >
              {{ t('announcement.management.create') }}
            </t-button>
          </div>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.keyword"
            clearable
            class="toolbar__search"
            :placeholder="t('announcement.management.filters.keyword')"
            type="search"
            @enter="handleSearch"
          />
          <t-select
            v-model="filters.status"
            clearable
            class="toolbar__select"
            :options="statusFilterOptions"
            :placeholder="t('announcement.management.filters.status')"
          />
          <t-select
            v-model="filters.level"
            clearable
            class="toolbar__select"
            :options="levelFilterOptions"
            :placeholder="t('announcement.management.filters.level')"
          />
          <t-select
            v-model="filters.pinned"
            clearable
            class="toolbar__select"
            :options="pinnedFilterOptions"
            :placeholder="t('announcement.management.filters.pinned')"
          />
          <t-select
            v-model="filters.sort"
            class="toolbar__select"
            :options="sortOptions"
            :placeholder="t('announcement.management.filters.sort')"
          />
          <t-button theme="default" variant="text" @click="resetFilters">
            {{ t('announcement.management.reset') }}
          </t-button>
        </template>
        <template #actions>
          <t-button theme="primary" variant="outline" :loading="loading" @click="handleSearch">
            {{ t('announcement.management.search') }}
          </t-button>
        </template>
      </management-toolbar>

      <management-table-card>
        <template #head>
          <div class="announcement-table-summary">
            <div>
              <p class="announcement-table-summary__count">
                {{ t('announcement.management.summary', { count: total }) }}
              </p>
              <p class="announcement-table-summary__hint">{{ t('announcement.management.tableHint') }}</p>
            </div>
            <t-button v-if="hasActiveFilters" theme="default" variant="text" @click="resetFilters">
              {{ t('announcement.management.reset') }}
            </t-button>
          </div>
        </template>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('announcement.management.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchAnnouncements">
              {{ t('announcement.management.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :data="presentedRows"
          :columns="columns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          cell-empty-content="-"
          @sort-change="handleSortChange"
        >
          <template #title="{ row }">
            <div class="announcement-title-cell">
              <strong>{{ row.title }}</strong>
              <span>{{ row.content }}</span>
            </div>
          </template>

          <template #status="{ row }">
            <t-tag :theme="row.statusTheme" variant="light">
              {{ row.statusLabel }}
            </t-tag>
          </template>

          <template #level="{ row }">
            <t-tag :theme="row.levelTheme" variant="light">
              {{ row.levelLabel }}
            </t-tag>
          </template>

          <template #delivery_mode="{ row }">
            <t-tag :theme="row.deliveryMode === 'popup' ? 'primary' : 'default'" variant="light">
              {{ row.deliveryModeLabel }}
            </t-tag>
          </template>

          <template #pinned="{ row }">
            <t-tag :theme="row.pinned ? 'primary' : 'default'" variant="light">
              {{ row.pinnedLabel }}
            </t-tag>
          </template>

          <template #publish_at="{ row }">
            <span class="table-muted">{{ row.publishAtLabel }}</span>
          </template>

          <template #expire_at="{ row }">
            <span class="table-muted">{{ row.expireAtLabel }}</span>
          </template>

          <template #updated_at="{ row }">
            <span class="table-muted">{{ row.updatedAtLabel }}</span>
          </template>

          <template #operation="{ row }">
            <table-action-menu
              :actions="rowActions(row)"
              :more-label="t('announcement.management.more')"
              :more-label-fallback="t('announcement.management.more')"
              @action="(action) => handleRowAction(action, row)"
            />
          </template>

          <template #empty>
            <div class="table-empty-state">
              <t-empty
                :title="t('announcement.management.emptyTitle')"
                :description="t('announcement.management.emptyDescription')"
              >
                <template #action>
                  <div class="table-empty-state__actions">
                    <t-button
                      v-if="hasActiveFilters"
                      theme="default"
                      variant="outline"
                      data-testid="announcement-empty-clear-filters"
                      @click="resetFilters"
                    >
                      {{ t('announcement.management.reset') }}
                    </t-button>
                    <t-button
                      v-permission="permissionCodes.CREATE"
                      theme="primary"
                      data-testid="announcement-empty-create"
                      @click="openCreateDrawer"
                    >
                      {{ t('announcement.management.emptyCreate') }}
                    </t-button>
                  </div>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>

        <template #footer>
          <management-table-pagination :summary="t('announcement.management.footerTotal', { count: total })">
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
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="formDrawerVisible"
      :header="formDrawerTitle"
      :footer="false"
      placement="right"
      size="620px"
      destroy-on-close
    >
      <t-form
        ref="formRef"
        class="announcement-form"
        :data="formState"
        :rules="formRules"
        label-align="top"
        @submit="handleFormSubmit"
      >
        <section class="drawer-section">
          <h3>{{ t('announcement.management.form.basicInfo') }}</h3>
          <t-form-item name="title" :label="t('announcement.management.form.title')">
            <t-input v-model="formState.title" :placeholder="t('announcement.management.form.titlePlaceholder')" />
          </t-form-item>
          <t-form-item name="content" :label="t('announcement.management.form.content')">
            <t-textarea
              v-model="formState.content"
              :autosize="{ minRows: 8, maxRows: 14 }"
              :placeholder="t('announcement.management.form.contentPlaceholder')"
            />
          </t-form-item>
          <t-form-item name="level" :label="t('announcement.management.form.level')">
            <t-select
              v-model="formState.level"
              :options="levelOptions"
              :placeholder="t('announcement.management.form.levelPlaceholder')"
            />
          </t-form-item>
          <t-form-item name="delivery_mode">
            <template #label>
              <span class="announcement-form__label-with-help">
                {{ t('announcement.management.form.deliveryMode') }}
                <t-tooltip
                  placement="top"
                  :content="t(`announcement.management.form.deliveryModeHelp.${formState.delivery_mode}`)"
                >
                  <t-icon class="announcement-form__help-icon" name="help-circle" />
                </t-tooltip>
              </span>
            </template>
            <t-select
              v-model="formState.delivery_mode"
              :options="deliveryModeOptions"
              :placeholder="t('announcement.management.form.deliveryModePlaceholder')"
            />
          </t-form-item>
          <t-form-item name="pinned">
            <t-checkbox v-model="formState.pinned">
              {{ t('announcement.management.form.pinned') }}
            </t-checkbox>
          </t-form-item>
          <section class="announcement-form__preview" aria-live="polite">
            <h4>{{ t('announcement.management.form.markdownPreview') }}</h4>
            <safe-markdown :source="formState.content" />
          </section>
        </section>

        <section class="drawer-section">
          <h3>{{ t('announcement.management.form.visibility') }}</h3>
          <t-form-item name="publish_at" :label="t('announcement.management.form.publishAt')">
            <t-date-picker
              v-model="formState.publish_at"
              clearable
              enable-time-picker
              value-type="YYYY-MM-DD HH:mm:ss"
              :placeholder="t('announcement.management.form.publishAtPlaceholder')"
            />
          </t-form-item>
          <t-form-item name="expire_at" :label="t('announcement.management.form.expireAt')">
            <t-date-picker
              v-model="formState.expire_at"
              clearable
              enable-time-picker
              value-type="YYYY-MM-DD HH:mm:ss"
              :placeholder="t('announcement.management.form.expireAtPlaceholder')"
            />
          </t-form-item>
        </section>

        <div class="drawer-actions">
          <t-button theme="default" variant="outline" @click="closeFormDrawer">
            {{ t('announcement.management.form.cancel') }}
          </t-button>
          <t-button theme="primary" type="submit" :loading="submitting">
            {{ t('announcement.management.form.confirm') }}
          </t-button>
        </div>
      </t-form>
    </t-drawer>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('announcement.management.detailDrawer.title')"
      :footer="false"
      placement="right"
      size="620px"
      destroy-on-close
    >
      <div v-if="detailRecord" class="announcement-detail">
        <section class="drawer-section">
          <h3>{{ t('announcement.management.detailDrawer.basic') }}</h3>
          <div class="detail-title-row">
            <strong>{{ detailRecord.title }}</strong>
            <div class="detail-tags">
              <t-tag :theme="detailRecord.statusTheme" variant="light">
                {{ detailRecord.statusLabel }}
              </t-tag>
              <t-tag :theme="detailRecord.levelTheme" variant="light">
                {{ detailRecord.levelLabel }}
              </t-tag>
              <t-tag :theme="detailRecord.deliveryMode === 'popup' ? 'primary' : 'default'" variant="light">
                {{ detailRecord.deliveryModeLabel }}
              </t-tag>
              <t-tag :theme="detailRecord.pinned ? 'primary' : 'default'" variant="light">
                {{ detailRecord.pinnedLabel }}
              </t-tag>
            </div>
          </div>
        </section>

        <section class="drawer-section">
          <h3>{{ t('announcement.management.detailDrawer.content') }}</h3>
          <safe-markdown class="announcement-detail__content" :source="detailRecord.content" />
        </section>

        <section class="drawer-section">
          <h3>{{ t('announcement.management.detailDrawer.timeline') }}</h3>
          <dl class="detail-list">
            <dt>{{ t('announcement.management.detailDrawer.publishAt') }}</dt>
            <dd>{{ detailRecord.publishAtLabel }}</dd>
            <dt>{{ t('announcement.management.detailDrawer.expireAt') }}</dt>
            <dd>{{ detailRecord.expireAtLabel }}</dd>
            <dt>{{ t('announcement.management.detailDrawer.createdAt') }}</dt>
            <dd>{{ detailRecord.createdAtLabel }}</dd>
            <dt>{{ t('announcement.management.detailDrawer.updatedAt') }}</dt>
            <dd>{{ detailRecord.updatedAtLabel }}</dd>
          </dl>
        </section>
      </div>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import type { FormRule, PageInfo, SortInfo, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  createActionColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  TableActionMenu,
} from '@/shared/components/management';
import { SafeMarkdown } from '@/shared/components/markdown';
import { isApiRequestError } from '@/utils/request';

import {
  archiveAnnouncement,
  createAnnouncement,
  deleteAnnouncement,
  getAnnouncement,
  getAnnouncements,
  publishAnnouncement,
  updateAnnouncement,
} from '../../api/announcement';
import { ANNOUNCEMENT_PERMISSION_CODE } from '../../contract/permissions';
import { emitAnnouncementChanged } from '../../contract/refresh';
import { type AnnouncementViewModel, presentAnnouncement } from '../../domain/announcement-presenter';
import type {
  AnnouncementDeliveryMode,
  AnnouncementFilterState,
  AnnouncementFormState,
  AnnouncementItem,
  AnnouncementLevel,
  AnnouncementPinnedFilter,
  AnnouncementStatus,
  AnnouncementStatusFilter,
  CreateAnnouncementRequest,
  UpdateAnnouncementRequest,
} from '../../types/announcement';

type RowAction = 'detail' | 'edit' | 'publish' | 'archive' | 'delete';
type FormMode = 'create' | 'edit';

type AnnouncementRowViewModel = AnnouncementViewModel & {
  raw: AnnouncementItem;
};

const { locale, t } = useI18n();
const permissionCodes = ANNOUNCEMENT_PERMISSION_CODE;

const loading = ref(false);
const submitting = ref(false);
const listError = ref('');
const rows = ref<AnnouncementItem[]>([]);
const total = ref(0);
const pagination = reactive({
  current: 1,
  pageSize: 20,
});
const filters = reactive<AnnouncementFilterState>({
  keyword: '',
  level: '',
  pinned: '',
  sort: 'updated_desc',
  status: '',
});

const formDrawerVisible = ref(false);
const formMode = ref<FormMode>('create');
const editingRecord = ref<AnnouncementItem | null>(null);
const formRef = ref<{ reset?: () => void } | null>(null);
const formState = reactive<AnnouncementFormState>(createEmptyFormState());
const detailDrawerVisible = ref(false);
const detailRecord = ref<AnnouncementViewModel | null>(null);

const statusValues: AnnouncementStatus[] = ['draft', 'published', 'archived'];
const levelValues: AnnouncementLevel[] = ['info', 'warning', 'success', 'error'];
const deliveryModeValues: AnnouncementDeliveryMode[] = ['silent', 'popup'];

const statusFilterOptions = computed(() =>
  statusValues.map((value) => ({
    label: t(`announcement.status.${value}`),
    value,
  })),
);
const levelOptions = computed(() =>
  levelValues.map((value) => ({
    label: t(`announcement.level.${value}`),
    value,
  })),
);
const levelFilterOptions = levelOptions;
const deliveryModeOptions = computed(() =>
  deliveryModeValues.map((value) => ({
    label: t(`announcement.deliveryMode.${value}`),
    value,
  })),
);
const pinnedFilterOptions = computed(() => [
  { label: t('announcement.pinned.yes'), value: 'true' },
  { label: t('announcement.pinned.no'), value: 'false' },
]);
const sortOptions = computed(() => [
  { label: t('announcement.management.sort.updatedDesc'), value: 'updated_desc' },
  { label: t('announcement.management.sort.publishDesc'), value: 'publish_desc' },
  { label: t('announcement.management.sort.pinnedPublishDesc'), value: 'pinned_publish_desc' },
]);

const hasActiveFilters = computed(
  () =>
    Boolean(filters.keyword.trim() || filters.status || filters.level || filters.pinned) ||
    filters.sort !== 'updated_desc',
);
const presentedRows = computed<AnnouncementRowViewModel[]>(() =>
  rows.value.map((item) => ({
    ...presentAnnouncement(item, t, locale.value),
    raw: item,
  })),
);
const columns = computed<TdBaseTableProps['columns']>(() => [
  createTextColumn(t('announcement.management.columns.title'), 'title', { minWidth: 240 }),
  createStatusColumn(t('announcement.management.columns.status'), 'status', 112),
  createStatusColumn(t('announcement.management.columns.level'), 'level', 104),
  createStatusColumn(t('announcement.management.columns.deliveryMode'), 'delivery_mode', 120),
  createStatusColumn(t('announcement.management.columns.pinned'), 'pinned', 104),
  createTimeColumn(t('announcement.management.columns.publishAt'), 'publish_at', 168),
  createTimeColumn(t('announcement.management.columns.expireAt'), 'expire_at', 168),
  {
    ...createTimeColumn(t('announcement.management.columns.updatedAt'), 'updated_at', 168),
    sorter: true,
    sortType: 'all',
  },
  createActionColumn(t('announcement.management.columns.operation'), 132),
]);
const tableContentWidth = computed(() => '1200');
const formDrawerTitle = computed(() =>
  formMode.value === 'create'
    ? t('announcement.management.form.createTitle')
    : t('announcement.management.form.editTitle'),
);
const formRules = computed<Record<keyof AnnouncementFormState, FormRule[]>>(() => ({
  content: [{ required: true, message: t('announcement.management.form.required.content'), type: 'error' }],
  delivery_mode: [{ required: true, message: t('announcement.management.form.required.deliveryMode'), type: 'error' }],
  expire_at: [
    { validator: validateExpireAt, message: t('announcement.management.form.invalidTimeWindow'), type: 'error' },
  ],
  level: [{ required: true, message: t('announcement.management.form.required.level'), type: 'error' }],
  pinned: [],
  publish_at: [],
  title: [{ required: true, message: t('announcement.management.form.required.title'), type: 'error' }],
}));

onMounted(() => {
  void fetchAnnouncements();
});

watch(
  () => [filters.status, filters.level, filters.pinned, filters.sort],
  () => {
    pagination.current = 1;
    void fetchAnnouncements();
  },
);

watch(
  () => [pagination.current, pagination.pageSize],
  () => {
    void fetchAnnouncements();
  },
);

async function fetchAnnouncements() {
  loading.value = true;
  listError.value = '';

  try {
    const page = await getAnnouncements({
      keyword: filters.keyword.trim() || undefined,
      level: normalizeLevelFilter(filters.level),
      page: pagination.current,
      page_size: pagination.pageSize,
      pinned: normalizePinnedFilter(filters.pinned),
      sort: filters.sort,
      status: normalizeStatusFilter(filters.status),
    });
    rows.value = page.items;
    total.value = page.total;
    pagination.current = page.page || pagination.current;
    pagination.pageSize = page.page_size || pagination.pageSize;
  } catch (error) {
    listError.value = readableError(error, t('announcement.management.loadFailed'));
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.current = 1;
  void fetchAnnouncements();
}

function resetFilters() {
  filters.keyword = '';
  filters.level = '';
  filters.pinned = '';
  filters.sort = 'updated_desc';
  filters.status = '';
  pagination.current = 1;
  void fetchAnnouncements();
}

function handlePageChange(pageInfo: PageInfo) {
  pagination.current = pageInfo.current;
  pagination.pageSize = pageInfo.pageSize;
}

function handleSortChange(sort: SortInfo | SortInfo[]) {
  const nextSort = Array.isArray(sort) ? sort[0] : sort;
  filters.sort = nextSort?.descending === false ? 'publish_desc' : 'updated_desc';
}

function openCreateDrawer() {
  formMode.value = 'create';
  editingRecord.value = null;
  Object.assign(formState, createEmptyFormState());
  formDrawerVisible.value = true;
}

function openEditDrawer(row: AnnouncementRowViewModel) {
  formMode.value = 'edit';
  editingRecord.value = row.raw;
  Object.assign(formState, toFormState(row.raw));
  formDrawerVisible.value = true;
}

function closeFormDrawer() {
  formDrawerVisible.value = false;
}

async function openDetailDrawer(row: AnnouncementRowViewModel) {
  try {
    const item = await getAnnouncement(row.id);
    detailRecord.value = presentAnnouncement(item, t, locale.value);
    detailDrawerVisible.value = true;
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.detailLoadFailed')));
  }
}

async function handleFormSubmit(context: SubmitContext) {
  if (context.validateResult !== true || !isTimeWindowValid(formState)) {
    if (!isTimeWindowValid(formState)) {
      MessagePlugin.error(t('announcement.management.form.invalidTimeWindow'));
    }
    return;
  }

  submitting.value = true;
  try {
    const payload = toMutationPayload(formState);
    if (formMode.value === 'create') {
      await createAnnouncement(payload);
      MessagePlugin.success(t('announcement.management.createSuccess'));
    } else if (editingRecord.value) {
      await updateAnnouncement(editingRecord.value.id, payload);
      MessagePlugin.success(t('announcement.management.updateSuccess'));
    }

    closeFormDrawer();
    await fetchAnnouncements();
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.submitFailed')));
  } finally {
    submitting.value = false;
  }
}

function rowActions(row: AnnouncementRowViewModel) {
  return [
    {
      label: 'announcement.management.detail',
      value: 'detail',
    },
    {
      disabled: row.status === 'archived',
      label: 'announcement.management.edit',
      value: 'edit',
    },
    {
      disabled: row.status === 'published',
      label: 'announcement.management.publishNow',
      value: 'publish',
    },
    {
      disabled: row.status !== 'published',
      label: 'announcement.management.archive',
      value: 'archive',
    },
    {
      label: 'announcement.management.delete',
      value: 'delete',
    },
  ];
}

function handleRowAction(action: string, row: AnnouncementRowViewModel) {
  switch (action as RowAction) {
    case 'detail':
      void openDetailDrawer(row);
      break;
    case 'edit':
      openEditDrawer(row);
      break;
    case 'publish':
      void publishRow(row);
      break;
    case 'archive':
      void archiveRow(row);
      break;
    case 'delete':
      void deleteRow(row);
      break;
    default:
      break;
  }
}

async function publishRow(row: AnnouncementRowViewModel) {
  try {
    await publishAnnouncement(row.id);
    MessagePlugin.success(t('announcement.management.publishSuccess'));
    await fetchAnnouncements();
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.publishFailed')));
  }
}

async function archiveRow(row: AnnouncementRowViewModel) {
  try {
    await archiveAnnouncement(row.id);
    MessagePlugin.success(t('announcement.management.archiveSuccess'));
    await fetchAnnouncements();
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.archiveFailed')));
  }
}

async function deleteRow(row: AnnouncementRowViewModel) {
  if (!window.confirm(t('announcement.management.deleteConfirm'))) {
    return;
  }

  try {
    await deleteAnnouncement(row.id);
    MessagePlugin.success(t('announcement.management.deleteSuccess'));
    if (detailRecord.value?.id === row.id) {
      detailDrawerVisible.value = false;
      detailRecord.value = null;
    }
    await fetchAnnouncements();
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.deleteFailed')));
  }
}

function normalizeStatusFilter(value: AnnouncementStatusFilter) {
  return value || undefined;
}

function normalizeLevelFilter(value: '' | AnnouncementLevel) {
  return value || undefined;
}

function normalizePinnedFilter(value: AnnouncementPinnedFilter) {
  if (value === 'true') return true;
  if (value === 'false') return false;
  return undefined;
}

function createEmptyFormState(): AnnouncementFormState {
  return {
    content: '',
    delivery_mode: 'silent',
    expire_at: '',
    level: 'info',
    pinned: false,
    publish_at: '',
    title: '',
  };
}

function toFormState(item: AnnouncementItem): AnnouncementFormState {
  return {
    content: item.content,
    delivery_mode: item.delivery_mode,
    expire_at: toDatePickerValue(item.expire_at),
    level: item.level,
    pinned: item.pinned,
    publish_at: toDatePickerValue(item.publish_at),
    title: item.title,
  };
}

function toMutationPayload(state: AnnouncementFormState): CreateAnnouncementRequest | UpdateAnnouncementRequest {
  return {
    content: state.content.trim(),
    delivery_mode: state.delivery_mode,
    expire_at: toApiDateTime(state.expire_at),
    level: state.level,
    pinned: state.pinned,
    publish_at: toApiDateTime(state.publish_at),
    title: state.title.trim(),
  };
}

function validateExpireAt() {
  return isTimeWindowValid(formState);
}

function isTimeWindowValid(state: AnnouncementFormState) {
  if (!state.publish_at || !state.expire_at) {
    return true;
  }

  return new Date(state.expire_at).getTime() > new Date(state.publish_at).getTime();
}

function toDatePickerValue(value?: string | null) {
  return value
    ? value
        .replace('T', ' ')
        .replace(/\.\d+Z$/u, '')
        .replace(/Z$/u, '')
    : '';
}

function toApiDateTime(value: string) {
  if (!value) {
    return null;
  }

  const normalized = value.includes('T') ? value : value.replace(' ', 'T');
  const date = new Date(normalized);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date.toISOString();
}

function readableError(error: unknown, fallback: string) {
  if (isApiRequestError(error)) {
    return error.message || fallback;
  }

  return error instanceof Error && error.message ? error.message : fallback;
}
</script>
<style scoped lang="less">
.announcement-management-page {
  min-width: 0;
}

.announcement-management-page__header-actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
}

.toolbar__search {
  width: min(320px, 100%);
}

.toolbar__select {
  width: 176px;
}

.announcement-table-summary {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.announcement-table-summary > div {
  min-width: 0;
}

.announcement-table-summary__count {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.announcement-table-summary__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-6) 0 0;
}

.announcement-title-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.announcement-title-cell strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.announcement-title-cell span,
.table-muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.announcement-title-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-empty-state {
  padding: var(--graft-density-gap-24) 0;
}

.table-empty-state__actions,
.drawer-actions,
.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
}

.announcement-form,
.announcement-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.drawer-section {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-16);
}

.drawer-section h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0 0 var(--graft-density-gap-14);
}

.announcement-form__label-with-help {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
}

.announcement-form__help-icon {
  color: var(--td-text-color-secondary);
  cursor: help;
  font-size: var(--td-font-size-body-medium);
}

.announcement-form__preview {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-12);
}

.announcement-form__preview h4 {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0 0 var(--graft-density-gap-10);
}

.drawer-actions {
  justify-content: flex-end;
  padding-top: var(--graft-density-gap-4);
}

.detail-title-row {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.detail-title-row strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.announcement-detail__content {
  max-height: 420px;
  overflow: auto;
}

.detail-list {
  display: grid;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-16);
  grid-template-columns: max-content minmax(0, 1fr);
  margin: 0;
}

.detail-list dt {
  color: var(--td-text-color-secondary);
}

.detail-list dd {
  color: var(--td-text-color-primary);
  margin: 0;
}

@media (width <= 768px) {
  .toolbar__search,
  .toolbar__select {
    width: 100%;
  }

  .announcement-table-summary {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--graft-density-gap-10);
  }

  .announcement-management-page__header-actions {
    justify-content: flex-start;
    width: 100%;
  }
}
</style>
