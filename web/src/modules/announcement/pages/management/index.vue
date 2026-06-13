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
              <t-tooltip placement="top-left" :content="row.summary">
                <span>{{ row.summary }}</span>
              </t-tooltip>
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
      placement="right"
      size="620px"
      destroy-on-close
      drawer-class-name="announcement-form-drawer"
    >
      <t-form
        id="announcement-form"
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
          <div class="announcement-form__preview-actions">
            <t-space break-line>
              <t-button theme="primary" variant="outline" @click="toggleInlinePreview">
                {{
                  inlinePreviewVisible
                    ? t('announcement.management.form.collapsePreview')
                    : t('announcement.management.form.previewCurrent')
                }}
              </t-button>
              <t-button theme="default" variant="outline" @click="openFullPreview">
                {{ t('announcement.management.form.openFullPreview') }}
              </t-button>
            </t-space>
          </div>
          <section
            v-if="inlinePreviewVisible"
            class="announcement-form__inline-preview"
            :aria-label="t('announcement.management.form.markdownPreview')"
            aria-live="polite"
          >
            <template v-if="hasPreviewContent">
              <markdown-viewer :source="formState.content" />
            </template>
            <t-empty v-else :description="t('announcement.management.form.emptyPreview')" />
          </section>
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
        </section>

        <section class="drawer-section">
          <h3>{{ t('announcement.management.form.visibility') }}</h3>
          <t-form-item name="publish_at" :label="t('announcement.management.form.publishAt')">
            <t-date-picker
              v-model="formState.publish_at"
              clearable
              enable-time-picker
              :default-time="currentTimeDefault"
              value-type="YYYY-MM-DD HH:mm:ss"
              :placeholder="t('announcement.management.form.publishAtPlaceholder')"
            />
          </t-form-item>
          <t-form-item name="expire_at" :label="t('announcement.management.form.expireAt')">
            <t-date-picker
              v-model="formState.expire_at"
              clearable
              enable-time-picker
              :default-time="currentTimeDefault"
              value-type="YYYY-MM-DD HH:mm:ss"
              :placeholder="t('announcement.management.form.expireAtPlaceholder')"
            />
          </t-form-item>
        </section>
      </t-form>
      <template #footer>
        <div class="drawer-actions">
          <t-button theme="default" variant="outline" @click="closeFormDrawer">
            {{ t('announcement.management.form.cancel') }}
          </t-button>
          <t-button theme="primary" :loading="submitting" @click="submitForm">
            {{ t('announcement.management.form.confirm') }}
          </t-button>
        </div>
      </template>
    </t-drawer>

    <t-dialog
      v-model:visible="fullPreviewVisible"
      :aria-label="t('announcement.management.form.markdownPreview')"
      :header="false"
      :confirm-btn="null"
      :cancel-btn="t('announcement.management.form.closePreview')"
      placement="center"
      width="min(880px, calc(100vw - 48px))"
      destroy-on-close
      dialog-class-name="announcement-preview-dialog"
    >
      <article class="announcement-preview-panel">
        <header class="announcement-preview-panel__header">
          <h2>{{ previewTitle }}</h2>
          <div class="detail-tags">
            <t-tag :theme="previewLevelTheme" variant="light">
              {{ previewLevelLabel }}
            </t-tag>
            <t-tag :theme="formState.delivery_mode === 'popup' ? 'primary' : 'default'" variant="light">
              {{ previewDeliveryModeLabel }}
            </t-tag>
          </div>
        </header>
        <div class="announcement-preview-panel__body">
          <template v-if="hasPreviewContent">
            <markdown-viewer :source="formState.content" />
          </template>
          <t-empty v-else :description="t('announcement.management.form.emptyPreview')" />
        </div>
      </article>
    </t-dialog>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('announcement.management.detailDrawer.title')"
      :footer="false"
      placement="right"
      size="min(880px, 72vw)"
      destroy-on-close
      drawer-class-name="announcement-detail-drawer"
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
          <markdown-viewer class="announcement-detail__content" :source="detailRecord.content" />
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

    <t-dialog
      v-model:visible="deleteDialogVisible"
      :header="t('announcement.management.deleteDialog.title')"
      :confirm-btn="deleteDialogConfirmBtn"
      :cancel-btn="t('announcement.management.deleteDialog.cancel')"
      :confirm-loading="deleting"
      theme="danger"
      width="420px"
      placement="center"
      destroy-on-close
      @confirm="confirmDelete"
      @close="closeDeleteDialog"
    >
      <p class="announcement-delete-dialog__description">
        {{ t('announcement.management.deleteDialog.description') }}
      </p>
      <p v-if="deleteTarget" class="announcement-delete-dialog__target">
        {{ deleteTarget.title }}
      </p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import type { ButtonProps, FormRule, PageInfo, SortInfo, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
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
import { MarkdownViewer } from '@/shared/components/markdown';
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
import {
  announcementLevelTheme,
  type AnnouncementViewModel,
  presentAnnouncement,
} from '../../domain/announcement-presenter';
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
const deleting = ref(false);
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
const formRef = ref<{ reset?: () => void; submit?: () => void } | null>(null);
const formState = reactive<AnnouncementFormState>(createEmptyFormState());
const inlinePreviewVisible = ref(false);
const fullPreviewVisible = ref(false);
const detailDrawerVisible = ref(false);
const detailRecord = ref<AnnouncementViewModel | null>(null);
const deleteDialogVisible = ref(false);
const deleteTarget = ref<AnnouncementRowViewModel | null>(null);
const currentTimeDefault = formatTimePart(new Date());

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
const deleteDialogConfirmBtn = computed<ButtonProps>(() => ({
  content: t('announcement.management.deleteDialog.confirm'),
  disabled: deleting.value,
  theme: 'danger',
}));
const formDrawerTitle = computed(() =>
  formMode.value === 'create'
    ? t('announcement.management.form.createTitle')
    : t('announcement.management.form.editTitle'),
);
const hasPreviewContent = computed(() => Boolean(formState.content.trim()));
const previewTitle = computed(() => formState.title.trim() || t('announcement.management.form.untitledPreview'));
const previewLevelLabel = computed(() => t(`announcement.level.${formState.level}`));
const previewLevelTheme = computed(() => announcementLevelTheme(formState.level));
const previewDeliveryModeLabel = computed(() => t(`announcement.deliveryMode.${formState.delivery_mode}`));
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
  () => [filters.level, filters.pinned, filters.sort, filters.status, pagination.current, pagination.pageSize],
  (_next, previous) => {
    const filtersChanged = Boolean(previous) && hasFilterStateChanged(previous);
    if (filtersChanged && pagination.current !== 1) {
      pagination.current = 1;
      return;
    }

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
  setCurrentPageAndMaybeFetch(1);
}

function resetFilters() {
  const shouldFetch =
    filters.keyword !== '' ||
    filters.level !== '' ||
    filters.pinned !== '' ||
    filters.sort !== 'updated_desc' ||
    filters.status !== '' ||
    pagination.current !== 1;
  filters.keyword = '';
  filters.level = '';
  filters.pinned = '';
  filters.sort = 'updated_desc';
  filters.status = '';
  if (shouldFetch) {
    setCurrentPageAndMaybeFetch(1);
  }
}

function handlePageChange(pageInfo: PageInfo) {
  pagination.current = pageInfo.current;
  pagination.pageSize = pageInfo.pageSize;
}

function handleSortChange(sort: SortInfo | SortInfo[]) {
  const nextSort = Array.isArray(sort) ? sort[0] : sort;
  filters.sort = nextSort?.descending === false ? 'publish_desc' : 'updated_desc';
}

function hasFilterStateChanged(previous: (string | number)[]) {
  return (
    filters.level !== previous[0] ||
    filters.pinned !== previous[1] ||
    filters.sort !== previous[2] ||
    filters.status !== previous[3]
  );
}

function setCurrentPageAndMaybeFetch(page: number) {
  if (pagination.current === page) {
    void fetchAnnouncements();
    return;
  }

  pagination.current = page;
}

function openCreateDrawer() {
  formMode.value = 'create';
  editingRecord.value = null;
  Object.assign(formState, createEmptyFormState());
  resetPreviewState();
  formDrawerVisible.value = true;
}

function openEditDrawer(row: AnnouncementRowViewModel) {
  formMode.value = 'edit';
  editingRecord.value = row.raw;
  Object.assign(formState, toFormState(row.raw));
  resetPreviewState();
  formDrawerVisible.value = true;
}

function closeFormDrawer() {
  formDrawerVisible.value = false;
  resetPreviewState();
}

function resetPreviewState() {
  inlinePreviewVisible.value = false;
  fullPreviewVisible.value = false;
}

function toggleInlinePreview() {
  inlinePreviewVisible.value = !inlinePreviewVisible.value;
}

function openFullPreview() {
  fullPreviewVisible.value = true;
}

function submitForm() {
  formRef.value?.submit?.();
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
  const actions = [
    {
      label: 'announcement.management.detail',
      value: 'detail',
    },
    {
      disabled: row.status === 'archived',
      label: 'announcement.management.edit',
      value: 'edit',
    },
  ];

  if (row.status === 'draft') {
    actions.push(
      {
        label: 'announcement.management.publishNow',
        value: 'publish',
      },
      {
        label: 'announcement.management.delete',
        value: 'delete',
      },
    );
  }

  if (row.status === 'published') {
    actions.push({
      label: 'announcement.management.archive',
      value: 'archive',
    });
  }

  if (row.status === 'archived') {
    actions.push(
      {
        label: 'announcement.management.republish',
        value: 'publish',
      },
      {
        label: 'announcement.management.delete',
        value: 'delete',
      },
    );
  }

  return actions;
}

function handleRowAction(action: string, row: AnnouncementRowViewModel) {
  if (action === 'edit' && row.status === 'archived') {
    return;
  }

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
    if (row.status === 'archived') {
      await publishAnnouncement(row.id, { publish_at: new Date().toISOString() });
    } else {
      await publishAnnouncement(row.id);
    }
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
  if (row.status === 'published') {
    MessagePlugin.warning(t('announcement.management.deletePublishedHint'));
    return;
  }

  deleteTarget.value = row;
  deleteDialogVisible.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) {
    return;
  }

  const target = deleteTarget.value;
  deleting.value = true;
  try {
    await deleteAnnouncement(target.id);
    MessagePlugin.success(t('announcement.management.deleteSuccess'));
    if (detailRecord.value?.id === target.id) {
      detailDrawerVisible.value = false;
      detailRecord.value = null;
    }
    deleteDialogVisible.value = false;
    deleteTarget.value = null;
    await fetchAnnouncements();
    emitAnnouncementChanged();
  } catch (error) {
    MessagePlugin.error(readableError(error, t('announcement.management.deleteFailed')));
  } finally {
    deleting.value = false;
  }
}

function closeDeleteDialog() {
  if (deleting.value) {
    return;
  }
  deleteDialogVisible.value = false;
  deleteTarget.value = null;
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

  const expireAt = parseDatePickerValue(state.expire_at);
  const publishAt = parseDatePickerValue(state.publish_at);
  if (!expireAt || !publishAt) {
    return true;
  }

  return expireAt.getTime() > publishAt.getTime();
}

function toDatePickerValue(value?: string | null) {
  if (!value) {
    return '';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }

  return formatDatePickerValue(date);
}

function toApiDateTime(value: string) {
  if (!value) {
    return null;
  }

  const date = parseDatePickerValue(value);
  if (!date) {
    return null;
  }

  return date.toISOString();
}

function parseDatePickerValue(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})$/u.exec(value.trim());
  if (!match) {
    const fallback = new Date(value);
    return Number.isNaN(fallback.getTime()) ? null : fallback;
  }

  const [, year, month, day, hour, minute, second] = match;
  return new Date(Number(year), Number(month) - 1, Number(day), Number(hour), Number(minute), Number(second));
}

function formatDatePickerValue(date: Date) {
  return `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())} ${formatTimePart(
    date,
  )}`;
}

function formatTimePart(date: Date) {
  return `${padDatePart(date.getHours())}:${padDatePart(date.getMinutes())}:${padDatePart(date.getSeconds())}`;
}

function padDatePart(value: number) {
  return String(value).padStart(2, '0');
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
  display: flex;
  justify-content: center;
  padding: var(--graft-density-gap-24) 0;
}

.table-empty-state__actions,
.drawer-actions,
.detail-tags,
.announcement-form__preview-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
}

.table-empty-state__actions {
  justify-content: center;
  width: 100%;
}

.announcement-delete-dialog__description,
.announcement-delete-dialog__target {
  margin: 0;
}

.announcement-delete-dialog__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.announcement-delete-dialog__target {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-12);
  overflow-wrap: anywhere;
}

.announcement-form,
.announcement-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

:deep(.announcement-form-drawer) {
  display: flex;
  flex-direction: column;
  max-height: 100vh;
}

:deep(.announcement-form-drawer .t-drawer__body) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

:deep(.announcement-form-drawer .t-drawer__footer) {
  background: var(--td-bg-color-container);
  border-top: 1px solid var(--td-component-stroke);
  flex: 0 0 auto;
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

.announcement-form__preview-actions {
  margin: calc(var(--graft-density-gap-8) * -1) 0 var(--graft-density-gap-16);
}

.announcement-form__inline-preview {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  max-height: 260px;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.announcement-form__inline-preview :deep(.t-empty) {
  padding: var(--graft-density-gap-16) 0;
}

.announcement-preview-panel {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-18);
  max-height: min(72vh, 720px);
  min-width: 0;
}

.announcement-preview-panel__header {
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding-bottom: var(--graft-density-gap-14);
}

.announcement-preview-panel__header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
  overflow-wrap: anywhere;
}

.announcement-preview-panel__body {
  min-height: 0;
  overflow: auto;
  padding-right: var(--graft-density-gap-4);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.announcement-preview-panel__body :deep(.t-empty) {
  padding: var(--graft-density-gap-24) 0;
}

.drawer-actions {
  justify-content: flex-end;
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
  :deep(.announcement-detail-drawer) {
    width: calc(100vw - 24px) !important;
  }

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
