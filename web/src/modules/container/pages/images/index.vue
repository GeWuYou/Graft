<template>
  <div class="docker-images-page" data-page-type="list-form-detail">
    <management-page-header
      title-key="container.images.title"
      :title="t('container.images.title')"
      description-key="container.images.description"
      :description="t('container.images.description')"
      :source="{ labelKey: 'container.images.eyebrow', fallback: t('container.images.eyebrow') }"
    >
      <template #actions>
        <t-space>
          <t-button variant="outline" :loading="query.isFetching.value" @click="refresh">
            {{ t('container.images.actions.refresh') }}
          </t-button>
          <t-button variant="outline" :loading="cleanupLoading" @click="openCleanup">
            {{ t('container.images.actions.cleanup') }}
          </t-button>
          <t-button theme="primary" @click="pullDrawerVisible = true">{{
            t('container.images.actions.pull')
          }}</t-button>
        </t-space>
      </template>
    </management-page-header>

    <div class="docker-images-metrics">
      <t-card v-for="metric in metrics" :key="metric.label" size="small" :bordered="false">
        <p class="docker-images-metric__label">{{ metric.label }}</p>
        <strong class="docker-images-metric__value">{{ metric.value }}</strong>
      </t-card>
    </div>

    <management-toolbar>
      <template #filters>
        <t-input
          v-model="keyword"
          class="management-list-search"
          clearable
          :placeholder="t('container.images.search')"
          @clear="clearKeyword"
          @enter="applyKeyword"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
      </template>
    </management-toolbar>

    <management-paged-table
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :columns="columns"
      :rows="images"
      :loading="query.isFetching.value"
      :total="total"
      :footer-summary="footerSummary"
      :empty-title="t('container.images.emptyTitle')"
      :empty-description="
        t(submittedKeyword ? 'container.images.filteredEmptyDescription' : 'container.images.emptyDescription')
      "
      :selected-row-keys="selectedRowKeys"
      @select-change="handleSelectChange"
    >
      <template #batch>
        <div v-if="selectedRowKeys.length" class="docker-images-batch-bar">
          <span>{{ t('container.images.batch.selected', { count: selectedRowKeys.length }) }}</span>
          <t-space size="small">
            <t-button size="small" theme="danger" variant="outline" :loading="batchRemoving" @click="openBatchRemove">
              {{ t('container.images.batch.remove') }}
            </t-button>
            <t-button size="small" variant="text" @click="clearSelection">
              {{ t('container.images.batch.cancelSelection') }}
            </t-button>
          </t-space>
        </div>
      </template>
      <template #feedback>
        <t-alert v-if="query.isError.value" theme="error" :message="t('container.images.loadFailed')" />
      </template>
      <template #empty>
        <t-empty
          :title="t('container.images.emptyTitle')"
          :description="
            t(submittedKeyword ? 'container.images.filteredEmptyDescription' : 'container.images.emptyDescription')
          "
        >
          <template #action>
            <t-button v-if="submittedKeyword" variant="outline" @click="clearKeyword">
              {{ t('container.images.clearFilter') }}
            </t-button>
          </template>
        </t-empty>
      </template>
      <template #tags="{ row }">
        <t-space size="small" break-line>
          <t-tag v-for="tag in imageTags(row)" :key="tag" size="small" variant="light-outline">{{ tag }}</t-tag>
          <span v-if="!imageTags(row).length" class="docker-images-muted">{{ t('container.images.untagged') }}</span>
        </t-space>
      </template>
      <template #id="{ row }"
        ><span class="docker-images-code">{{ shortId(row.id) }}</span></template
      >
      <template #size="{ row }">{{ formatBytes(row.size_bytes) }}</template>
      <template #containers="{ row }">
        <t-space v-if="row.container_references?.length" size="small" break-line>
          <t-tooltip v-for="container in row.container_references" :key="container.id" :content="container.id">
            <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
          </t-tooltip>
        </t-space>
        <t-tag v-else size="small" variant="light-outline">{{ t('container.images.unused') }}</t-tag>
      </template>
      <template #created_at="{ row }">{{ formatLocaleDateTime(row.created_at, locale) }}</template>
      <template #actions="{ row }">
        <table-action-menu
          :actions="[
            { value: 'detail', label: 'container.images.actions.detail' },
            { value: 'tag', label: 'container.images.actions.tag' },
            { value: 'remove', label: 'container.images.actions.remove' },
          ]"
          :more-label="t('container.images.actions.more')"
          @action="handleRowAction($event, row)"
        />
      </template>
    </management-paged-table>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('container.images.detail.title')"
      size="640px"
      placement="right"
    >
      <t-loading :loading="detailLoading">
        <t-descriptions v-if="selectedImage" bordered :column="1" size="small">
          <t-descriptions-item :label="t('container.images.fields.id')"
            ><span class="docker-images-code">{{ selectedImage.id }}</span></t-descriptions-item
          >
          <t-descriptions-item :label="t('container.images.fields.tags')">{{
            imageTags(selectedImage).join(', ') || t('container.images.untagged')
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.digests')">{{
            selectedImage.repository_digests.join(', ') || '-'
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.size')">{{
            formatBytes(selectedImage.size_bytes)
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.createdAt')">{{
            formatLocaleDateTime(selectedImage.created_at, locale)
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.containers')">
            <t-space v-if="selectedImage.container_references?.length" size="small" break-line>
              <t-tooltip
                v-for="container in selectedImage.container_references"
                :key="container.id"
                :content="container.id"
              >
                <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
              </t-tooltip>
            </t-space>
            <span v-else>{{ t('container.images.unused') }}</span>
          </t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.platform')"
            >{{ selectedImage.operating_system || '-' }} / {{ selectedImage.architecture || '-' }}</t-descriptions-item
          >
          <t-descriptions-item :label="t('container.images.fields.labels')">
            <t-space v-if="Object.keys(selectedImage.labels ?? {}).length" direction="vertical" size="small">
              <span v-for="(value, key) in selectedImage.labels" :key="key" class="docker-images-code"
                >{{ key }}={{ value }}</span
              >
            </t-space>
            <span v-else>-</span>
          </t-descriptions-item>
        </t-descriptions>
      </t-loading>
    </t-drawer>

    <t-dialog
      v-model:visible="tagDialogVisible"
      :header="t('container.images.tag.title')"
      :confirm-btn="t('container.images.actions.tag')"
      :confirm-loading="tagging"
      @confirm="submitTag"
    >
      <t-form label-align="top">
        <t-form-item :label="t('container.images.tag.repository')"
          ><t-input v-model="tagForm.repository"
        /></t-form-item>
        <t-form-item :label="t('container.images.tag.tag')"><t-input v-model="tagForm.tag" /></t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="removeDialogVisible"
      theme="danger"
      :header="t('container.images.remove.title')"
      :confirm-btn="removeConfirmButton"
      :confirm-loading="removing || batchRemoving"
      @confirm="submitRemove"
    >
      <p v-if="selectedImage">{{ t('container.images.remove.confirm', { image: shortId(selectedImage.id) }) }}</p>
      <template v-else>
        <p>{{ t('container.images.batch.confirm', { count: selectedRowKeys.length }) }}</p>
        <p v-if="selectedBatchReferences.length" class="docker-images-risk">
          {{ t('container.images.batch.inUse') }}
        </p>
        <t-space v-if="selectedBatchReferences.length" size="small" break-line>
          <t-tooltip v-for="container in selectedBatchReferences" :key="container.id" :content="container.id">
            <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
          </t-tooltip>
        </t-space>
        <t-checkbox v-if="selectedBatchReferences.length" v-model="forceRemove">
          {{ t('container.images.remove.force') }}
        </t-checkbox>
      </template>
      <p v-if="selectedImage?.container_references?.length" class="docker-images-risk">
        {{ t('container.images.remove.inUse') }}
      </p>
      <t-checkbox v-if="selectedImage?.container_references?.length" v-model="forceRemove">{{
        t('container.images.remove.force')
      }}</t-checkbox>
    </t-dialog>

    <t-dialog
      v-model:visible="cleanupDialogVisible"
      dialog-class-name="docker-images-cleanup-dialog"
      :dialog-style="cleanupDialogStyle"
      :header="false"
      width="760px"
      @confirm="submitCleanup"
    >
      <div class="docker-images-cleanup graft-scrollbar">
        <header class="docker-images-cleanup__header">
          <div class="docker-images-cleanup__header-icon"><delete-icon /></div>
          <div>
            <h2>{{ t('container.images.cleanup.title') }}</h2>
            <p>{{ t('container.images.cleanup.subtitle') }}</p>
          </div>
        </header>

        <t-loading :loading="cleanupLoading">
          <t-card v-if="cleanupImages.length" class="docker-images-cleanup-summary" :bordered="false">
            <div class="docker-images-cleanup-summary__stats">
              <div>
                <span class="docker-images-cleanup-summary__label">{{
                  t('container.images.cleanup.candidateCount')
                }}</span>
                <strong>{{ cleanupImages.length }}</strong>
                <span>{{ t('container.images.cleanup.imageUnit') }}</span>
              </div>
              <div>
                <span class="docker-images-cleanup-summary__label">{{
                  t('container.images.cleanup.releaseSize')
                }}</span>
                <strong>{{ formatBytes(cleanupTotalSize) }}</strong>
              </div>
              <div>
                <span class="docker-images-cleanup-summary__label">{{ t('container.images.cleanup.source') }}</span>
                <strong>{{ t('container.images.cleanup.sourceValue') }}</strong>
              </div>
            </div>
          </t-card>

          <t-alert
            v-if="cleanupImages.length"
            class="docker-images-cleanup-warning"
            theme="warning"
            :message="t('container.images.cleanup.warning')"
          />

          <section v-if="cleanupImages.length" class="docker-images-cleanup-preview">
            <div class="docker-images-cleanup-section-head">
              <h3>{{ t('container.images.cleanup.candidateTitle', { count: cleanupImages.length }) }}</h3>
              <t-space size="small">
                <span>{{ t('container.images.cleanup.selectedCount', { count: cleanupSelectedIds.length }) }}</span>
                <t-button v-if="cleanupSelectedIds.length" size="small" variant="text" @click="clearCleanupSelection">
                  {{ t('container.images.cleanup.clearSelection') }}
                </t-button>
              </t-space>
            </div>

            <div class="docker-images-cleanup-preview-body">
              <t-table
                class="docker-images-cleanup-table"
                :columns="cleanupColumns"
                :data="cleanupPreviewImages"
                row-key="id"
                size="small"
                table-layout="fixed"
                :selected-row-keys="cleanupSelectedIds"
                @select-change="handleCleanupSelectChange"
              >
                <template #image="{ row }">
                  <div class="docker-images-cleanup-image">
                    <image-icon class="docker-images-cleanup-image__icon" />
                    <span class="docker-images-cleanup-image__name">{{
                      imageTags(row).join(', ') || shortId(row.id)
                    }}</span>
                  </div>
                </template>
                <template #status>
                  <t-tag size="small" variant="light-outline">{{ t('container.images.unused') }}</t-tag>
                </template>
                <template #size="{ row }">{{ formatBytes(row.size_bytes) }}</template>
              </t-table>

              <div v-if="cleanupPreviewPageCount > 1" class="docker-images-cleanup-pager">
                <t-tooltip :content="t('container.images.cleanup.previousPage')">
                  <t-button
                    size="small"
                    variant="text"
                    shape="circle"
                    :disabled="cleanupPreviewPage === 1"
                    :aria-label="t('container.images.cleanup.previousPage')"
                    @click="previousCleanupPage"
                  >
                    <arrow-up-icon />
                  </t-button>
                </t-tooltip>
                <span>{{ cleanupPreviewPage }} / {{ cleanupPreviewPageCount }}</span>
                <t-tooltip :content="t('container.images.cleanup.nextPage')">
                  <t-button
                    size="small"
                    variant="text"
                    shape="circle"
                    :disabled="cleanupPreviewPage === cleanupPreviewPageCount"
                    :aria-label="t('container.images.cleanup.nextPage')"
                    @click="nextCleanupPage"
                  >
                    <arrow-down-icon />
                  </t-button>
                </t-tooltip>
              </div>
            </div>
          </section>
        </t-loading>

        <t-empty v-if="!cleanupLoading && !cleanupImages.length" :title="t('container.images.cleanup.empty')" />
      </div>
      <template #footer>
        <div class="docker-images-cleanup-footer">
          <div>
            <span>{{ t('container.images.cleanup.footerRelease') }}</span>
            <strong>{{ formatBytes(cleanupSelectedSize) }}</strong>
          </div>
          <t-space size="small">
            <t-button variant="outline" @click="cleanupDialogVisible = false">
              {{ t('container.images.cleanup.cancel') }}
            </t-button>
            <t-button
              theme="danger"
              :disabled="!cleanupSelectedIds.length"
              :loading="batchRemoving"
              @click="submitCleanup"
            >
              {{ t('container.images.cleanup.removeSelected', { count: cleanupSelectedIds.length }) }}
            </t-button>
          </t-space>
        </div>
      </template>
    </t-dialog>

    <t-drawer
      v-model:visible="pullDrawerVisible"
      :header="t('container.images.pull.title')"
      size="720px"
      placement="right"
      :close-btn="!pulling"
    >
      <t-form label-align="top" @submit.prevent="startPull">
        <t-form-item :label="t('container.images.pull.reference')"
          ><t-input v-model="pullReference" :disabled="pulling" :placeholder="t('container.images.pull.placeholder')"
        /></t-form-item>
        <t-space>
          <t-button theme="primary" :loading="pulling" @click="startPull">{{
            t('container.images.actions.pull')
          }}</t-button>
          <t-button v-if="pulling" theme="danger" variant="outline" @click="cancelPull">{{
            t('container.images.actions.cancelPull')
          }}</t-button>
        </t-space>
      </t-form>
      <log-viewer
        v-bind="pullLogViewerBindings"
        class="docker-images-pull-log"
        :entries="pullLogEntries"
        :content-version="pullLogVersion"
      />
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
// 镜像页面只管理当前 Docker runtime 的镜像快照；批量删除按服务端上限分块，并把传输失败归并为逐项结果以保留已完成块。
import { ArrowDownIcon, ArrowUpIcon, DeleteIcon, ImageIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onUnmounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementPagedTable,
  ManagementPageHeader,
  ManagementToolbar,
  TableActionMenu,
} from '@/shared/components/management';
import {
  formatBytes,
  formatLocaleDateTime,
  LogBatchBuffer,
  LogRingBuffer,
  LogViewer,
  type StructuredLogEntry,
} from '@/shared/observability';

import { type DockerImageRecord, getDockerImage, getDockerImages } from '../../api/container';
import {
  batchRemoveDockerImages,
  type DockerImageBatchResult,
  type DockerImagePullEvent,
  pullDockerImage,
  removeDockerImage,
  tagDockerImage,
} from '../../api/image-actions';
import { type DockerImageQueryState, useDockerImageQuery } from '../../shared/docker-image-queries';

type DockerImage = DockerImageRecord;

const { locale, t } = useI18n();
const pagination = reactive({ current: 1, pageSize: 20 });
const keyword = ref('');
const submittedKeyword = ref('');
const imageQuery = computed<DockerImageQueryState>(() => ({
  pageSize: pagination.pageSize,
  offset: (pagination.current - 1) * pagination.pageSize,
  keyword: submittedKeyword.value,
}));
const query = useDockerImageQuery(imageQuery);
const selectedImage = ref<DockerImage | null>(null);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const tagDialogVisible = ref(false);
const removeDialogVisible = ref(false);
const pullDrawerVisible = ref(false);
const tagging = ref(false);
const removing = ref(false);
const forceRemove = ref(false);
const selectedRowKeys = ref<Array<string | number>>([]);
const selectedImages = ref(new Map<string, DockerImage>());
const batchRemoving = ref(false);
const cleanupDialogVisible = ref(false);
const cleanupLoading = ref(false);
const cleanupImages = ref<DockerImage[]>([]);
const cleanupSelectedIds = ref<string[]>([]);
const cleanupPreviewPage = ref(1);
const cleanupPreviewLimit = 8;
const cleanupDialogStyle = { maxHeight: '70vh' };
const tagForm = reactive({ repository: '', tag: '' });
const pullReference = ref('');
const pulling = ref(false);
const pullLogEntries = ref<readonly StructuredLogEntry[]>([]);
const pullLogVersion = ref(0);
let pullController: AbortController | null = null;
const pullLogBuffer = new LogRingBuffer<StructuredLogEntry>(1000);
const pullLogBatcher = new LogBatchBuffer<StructuredLogEntry>({ onFlush: commitPullLog });
const tagTarget = computed(() => {
  const repository = tagForm.repository.trim();
  const tag = tagForm.tag.trim();
  return repository && tag ? `${repository}:${tag}` : '';
});
const pullLogViewerBindings = computed(() => ({
  allLevelsLabel: pullLogLabel('allLevels'),
  autoScrollLabel: pullLogLabel('autoScroll'),
  autoScrollTooltipLabel: pullLogLabel('autoScrollTooltip'),
  basicInfoLabel: pullLogLabel('basicInfo'),
  clearLabel: pullLogLabel('clear'),
  collapseDetailLabel: pullLogLabel('collapseDetail'),
  copyErrorLabel: pullLogLabel('copyError'),
  copyJsonLabel: pullLogLabel('copyJson'),
  copyLabel: pullLogLabel('copy'),
  copyLineLabel: pullLogLabel('copyLine'),
  copyMessageLabel: pullLogLabel('copyMessage'),
  copySuccessLabel: pullLogLabel('copySuccess'),
  detailTitleLabel: pullLogLabel('detailTitle'),
  downloadLabel: pullLogLabel('download'),
  emptyLabel: t('container.images.pull.emptyLog'),
  importantFieldsLabel: pullLogLabel('importantFields'),
  jumpBottomLabel: pullLogLabel('jumpBottom'),
  levelFilterLabel: pullLogLabel('levelFilter'),
  levelLabel: pullLogLabel('level'),
  matchCountLabel: pullLogLabel('matchCount'),
  messageLabel: pullLogLabel('message'),
  metadataLabel: pullLogLabel('metadata'),
  operationLabel: pullLogLabel('operation'),
  pauseLabel: pullLogLabel('pause'),
  rawLabel: pullLogLabel('raw'),
  reconnectLabel: pullLogLabel('reconnect'),
  resumeLabel: pullLogLabel('resume'),
  retryLabel: pullLogLabel('retry'),
  searchPlaceholder: pullLogLabel('search'),
  sourceLabel: pullLogLabel('source'),
  stderrLabel: pullLogLabel('stderr'),
  stdoutLabel: pullLogLabel('stdout'),
  streamLabel: pullLogLabel('stream'),
  timeLabel: pullLogLabel('time'),
  truncatedLabel: pullLogLabel('truncated'),
  viewDetailLabel: pullLogLabel('viewDetail'),
  wrapLabel: pullLogLabel('wrap'),
}));

function pullLogLabel(key: string) {
  return t(`task.logs.${key}`);
}

const images = computed(() => query.data.value?.items ?? []);
const total = computed(() => query.data.value?.total ?? 0);
const summary = computed(() => query.data.value?.summary);
const metrics = computed(() => [
  { label: t('container.images.metrics.total'), value: summary.value ? summary.value.total : '--' },
  {
    label: t('container.images.metrics.size'),
    value: summary.value ? formatBytes(summary.value.size_bytes) : '--',
  },
  { label: t('container.images.metrics.inUse'), value: summary.value ? summary.value.in_use : '--' },
  { label: t('container.images.metrics.dangling'), value: summary.value ? summary.value.dangling : '--' },
]);
const footerSummary = computed(() => {
  if (!total.value) return t('container.images.pagination.empty');
  const start = (pagination.current - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.current * pagination.pageSize, total.value);
  return t('container.images.pagination.summary', { start, end, total: total.value });
});
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'tags', title: t('container.images.fields.tags'), ellipsis: true, minWidth: 260 },
  { colKey: 'id', title: t('container.images.fields.id'), width: 150 },
  { colKey: 'size', title: t('container.images.fields.size'), width: 120 },
  { colKey: 'containers', title: t('container.images.fields.containers'), minWidth: 220 },
  { colKey: 'created_at', title: t('container.images.fields.createdAt'), width: 180 },
  { colKey: 'actions', title: t('container.images.fields.actions'), width: 150, fixed: 'right' as const },
]);
const selectedBatchReferences = computed(() =>
  selectedRowKeys.value.flatMap((key) => selectedImages.value.get(String(key))?.container_references ?? []),
);
const removeConfirmButton = computed(() => ({
  content: t('container.images.actions.remove'),
  disabled:
    (!selectedImage.value && !selectedRowKeys.value.length) ||
    (!forceRemove.value &&
      (Boolean(selectedImage.value?.container_references?.length) || Boolean(selectedBatchReferences.value.length))),
}));
const cleanupSelectedSize = computed(() => {
  const selected = new Set(cleanupSelectedIds.value);
  return cleanupImages.value.reduce((total, image) => (selected.has(image.id) ? total + image.size_bytes : total), 0);
});
const cleanupTotalSize = computed(() => cleanupImages.value.reduce((total, image) => total + image.size_bytes, 0));
const cleanupPreviewPageCount = computed(() =>
  Math.max(1, Math.ceil(cleanupImages.value.length / cleanupPreviewLimit)),
);
const cleanupPreviewImages = computed(() =>
  cleanupImages.value.slice(
    (cleanupPreviewPage.value - 1) * cleanupPreviewLimit,
    cleanupPreviewPage.value * cleanupPreviewLimit,
  ),
);
const cleanupColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'image', title: t('container.images.cleanup.imageColumn'), ellipsis: true, minWidth: 280 },
  { colKey: 'status', title: t('container.images.cleanup.statusColumn'), width: 92 },
  { colKey: 'size', title: t('container.images.cleanup.sizeColumn'), width: 112, align: 'right' as const },
]);

function imageTags(image: DockerImage) {
  return image.repository_tags.filter(Boolean);
}
function shortId(value: string) {
  return value.replace(/^sha256:/, '').slice(0, 12);
}
function refresh() {
  return query.refetch();
}
function applyKeyword() {
  submittedKeyword.value = keyword.value.trim();
  pagination.current = 1;
}
function clearKeyword() {
  keyword.value = '';
  submittedKeyword.value = '';
  pagination.current = 1;
}
function handleSelectChange(rowKeys: Array<string | number>) {
  images.value.forEach((image) => selectedImages.value.set(image.id, image));
  const currentPageIds = new Set(images.value.map((image) => image.id));
  const preserved = selectedRowKeys.value.filter((key) => !currentPageIds.has(String(key)));
  selectedRowKeys.value = [...preserved, ...rowKeys.filter((key) => currentPageIds.has(String(key)))];
}
function clearSelection() {
  selectedRowKeys.value = [];
  selectedImages.value.clear();
}
function handleRowAction(action: string, image: DockerImage) {
  if (action === 'detail') openDetail(image);
  if (action === 'tag') openTag(image);
  if (action === 'remove') openRemove(image);
}
async function openDetail(image: DockerImage) {
  selectedImage.value = image;
  detailDrawerVisible.value = true;
  detailLoading.value = true;
  try {
    selectedImage.value = await getDockerImage(image.id);
  } catch {
    MessagePlugin.error(t('container.images.detail.loadFailed'));
  } finally {
    detailLoading.value = false;
  }
}
function openTag(image: DockerImage) {
  selectedImage.value = image;
  const reference = imageTags(image)[0] ?? '';
  const lastSlash = reference.lastIndexOf('/');
  const lastColon = reference.lastIndexOf(':');
  tagForm.repository = lastColon > lastSlash ? reference.slice(0, lastColon) : reference;
  tagForm.tag = 'latest';
  tagDialogVisible.value = true;
}
async function submitTag() {
  if (!selectedImage.value || !tagTarget.value) return;
  tagging.value = true;
  try {
    await tagDockerImage(selectedImage.value.id, { target: tagTarget.value });
    tagDialogVisible.value = false;
    await refresh();
    MessagePlugin.success(t('container.images.tag.success'));
  } catch {
    MessagePlugin.error(t('container.images.tag.failed'));
  } finally {
    tagging.value = false;
  }
}
function openRemove(image: DockerImage) {
  selectedImage.value = image;
  forceRemove.value = false;
  removeDialogVisible.value = true;
}
async function submitRemove() {
  if (!selectedImage.value) {
    await submitBatchRemove(selectedRowKeys.value.map(String), forceRemove.value);
    return;
  }
  if (selectedImage.value.container_references?.length && !forceRemove.value) return;
  removing.value = true;
  try {
    await removeDockerImage(selectedImage.value.id, { force: forceRemove.value });
    removeDialogVisible.value = false;
    forgetSelectedImages([selectedImage.value.id]);
    await refresh();
    MessagePlugin.success(t('container.images.remove.success'));
  } catch {
    MessagePlugin.error(t('container.images.remove.failed'));
  } finally {
    removing.value = false;
  }
}
function openBatchRemove() {
  if (!selectedRowKeys.value.length) return;
  forceRemove.value = false;
  selectedImage.value = null;
  removeDialogVisible.value = true;
}
function forgetSelectedImages(ids: string[]) {
  const removedIds = new Set(ids);
  selectedRowKeys.value = selectedRowKeys.value.filter((key) => !removedIds.has(String(key)));
  ids.forEach((id) => selectedImages.value.delete(id));
}
async function removeImageIds(ids: string[], force = false) {
  const results: DockerImageBatchResult['items'] = [];
  let hasUnknownResponse = false;
  for (let index = 0; index < ids.length; index += 100) {
    const chunkIds = ids.slice(index, index + 100);
    try {
      const response = await batchRemoveDockerImages({ ids: chunkIds, force });
      results.push(...response.items);
    } catch {
      hasUnknownResponse = true;
      results.push(...chunkIds.map((id) => ({ id, success: false, error_code: 'client_request_failed' })));
    }
  }
  return { hasUnknownResponse, items: results };
}
async function submitCleanup() {
  if (!cleanupSelectedIds.value.length) return;
  await submitBatchRemove(cleanupSelectedIds.value, false, true);
}
async function submitBatchRemove(ids: string[], force: boolean, cleanup = false) {
  batchRemoving.value = true;
  try {
    const { hasUnknownResponse, items } = await removeImageIds(ids, force);
    const successfulIds = new Set(items.filter((item) => item.success).map((item) => item.id));
    forgetSelectedImages([...successfulIds]);
    if (cleanup) {
      cleanupSelectedIds.value = cleanupSelectedIds.value.filter((id) => !successfulIds.has(id));
      cleanupImages.value = cleanupImages.value.filter((image) => !successfulIds.has(image.id));
      cleanupPreviewPage.value = Math.min(cleanupPreviewPage.value, cleanupPreviewPageCount.value);
      if (hasUnknownResponse) await reconcileCleanupCandidates(successfulIds);
    }
    const failed = items.filter((item) => !item.success);
    if (!failed.length) MessagePlugin.success(t('container.images.batch.success', { count: items.length }));
    else if (failed.length < items.length)
      MessagePlugin.warning(
        t('container.images.batch.partial', { success: items.length - failed.length, failed: failed.length }),
      );
    else MessagePlugin.error(t('container.images.batch.failed'));
    if (!failed.length || failed.length < items.length) {
      removeDialogVisible.value = false;
      if (!cleanup || !hasUnknownResponse) cleanupDialogVisible.value = false;
    }
    await refresh();
  } catch {
    MessagePlugin.error(t('container.images.batch.failed'));
  } finally {
    batchRemoving.value = false;
  }
}
async function openCleanup() {
  cleanupDialogVisible.value = true;
  cleanupLoading.value = true;
  cleanupImages.value = [];
  cleanupSelectedIds.value = [];
  cleanupPreviewPage.value = 1;
  try {
    const all = await fetchCleanupCandidates();
    cleanupImages.value = all;
    cleanupSelectedIds.value = all.map((image) => image.id);
  } catch {
    MessagePlugin.error(t('container.images.cleanup.loadFailed'));
  } finally {
    cleanupLoading.value = false;
  }
}
async function fetchCleanupCandidates() {
  const firstPage = await getDockerImages({ limit: 100, offset: 0, unused: true });
  const all = [...firstPage.items];
  for (let offset = firstPage.items.length; offset < firstPage.total; offset += 100) {
    const page = await getDockerImages({ limit: 100, offset, unused: true });
    all.push(...page.items);
  }
  return all;
}
async function reconcileCleanupCandidates(confirmedSuccessfulIds: Set<string>) {
  try {
    const candidates = await fetchCleanupCandidates();
    const candidateIds = new Set(candidates.map((image) => image.id));
    cleanupImages.value = candidates;
    cleanupSelectedIds.value = cleanupSelectedIds.value.filter(
      (id) => candidateIds.has(id) && !confirmedSuccessfulIds.has(id),
    );
    cleanupPreviewPage.value = Math.min(cleanupPreviewPage.value, cleanupPreviewPageCount.value);
  } catch {
    MessagePlugin.error(t('container.images.cleanup.loadFailed'));
  }
}
function clearCleanupSelection() {
  cleanupSelectedIds.value = [];
}
function handleCleanupSelectChange(rowKeys: Array<string | number>) {
  const currentPageIds = new Set(cleanupPreviewImages.value.map((image) => image.id));
  const preserved = cleanupSelectedIds.value.filter((id) => !currentPageIds.has(id));
  cleanupSelectedIds.value = [...preserved, ...rowKeys.filter((key) => currentPageIds.has(String(key))).map(String)];
}
function previousCleanupPage() {
  cleanupPreviewPage.value = Math.max(1, cleanupPreviewPage.value - 1);
}
function nextCleanupPage() {
  cleanupPreviewPage.value = Math.min(cleanupPreviewPageCount.value, cleanupPreviewPage.value + 1);
}
async function startPull() {
  if (!pullReference.value.trim() || pulling.value) return;
  pullLogBuffer.clear();
  pullLogEntries.value = [];
  pullLogVersion.value = 0;
  pulling.value = true;
  pullController = new AbortController();
  let pullCompleted = false;
  let pullEventFailed = false;
  try {
    await pullDockerImage({ reference: pullReference.value.trim() }, pullController.signal, (event) => {
      appendPullEvent(event);
      if (event.error) {
        pullEventFailed = true;
        throw new Error(event.status || 'Docker image pull failed.');
      }
      if (event.status === 'completed') pullCompleted = true;
    });
    if (!pullCompleted) throw new Error('Docker image pull did not reach a terminal state.');
    pullLogBatcher.flush();
    await refresh();
    MessagePlugin.success(t('container.images.pull.success'));
  } catch (error) {
    if (!pullController?.signal.aborted) {
      if (!pullEventFailed) {
        appendPullEvent({ error: true, status: error instanceof Error ? error.message : String(error) });
      }
      pullLogBatcher.flush();
      MessagePlugin.error(t('container.images.pull.failed'));
    }
  } finally {
    pulling.value = false;
    pullController = null;
  }
}
function appendPullEvent(event: DockerImagePullEvent) {
  const line = [event.id, event.status, event.progress].filter(Boolean).join(' ') || JSON.stringify(event);
  pullLogBatcher.append({
    line,
    occurredAt: new Date().toISOString(),
    stream: event.error ? 'stderr' : 'stdout',
    ...(event.error ? { level: 'error' as const } : {}),
  });
}
function commitPullLog(entries: readonly StructuredLogEntry[]) {
  for (const entry of entries) pullLogBuffer.append(entry);
  const view = pullLogBuffer.snapshot();
  pullLogEntries.value = view.toArray();
  pullLogVersion.value = view.version;
}
function cancelPull() {
  pullController?.abort();
}
onUnmounted(() => {
  pullController?.abort();
  pullLogBatcher.destroy();
});
</script>
<style scoped lang="less">
.docker-images-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.docker-images-metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.docker-images-muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-metric__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0 0 var(--graft-density-gap-4);
}

.docker-images-metric__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.docker-images-code {
  font-family: var(--td-font-family-medium);
  overflow-wrap: anywhere;
}

.docker-images-risk {
  color: var(--td-warning-color-7);
}

.docker-images-batch-bar {
  align-items: center;
  display: flex;
  justify-content: space-between;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-16);
}

.docker-images-cleanup {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  max-height: calc(70vh - 120px);
  overflow: auto;
}

.docker-images-cleanup__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
}

.docker-images-cleanup__header h2,
.docker-images-cleanup__header p,
.docker-images-cleanup-section-head h3,
.docker-images-cleanup-section-head span {
  margin: 0;
}

.docker-images-cleanup__header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.docker-images-cleanup__header p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.docker-images-cleanup__header-icon {
  align-items: center;
  background: var(--td-brand-color-1);
  border-radius: var(--td-radius-medium);
  color: var(--td-brand-color-7);
  display: flex;
  flex: 0 0 40px;
  height: 40px;
  justify-content: center;
  width: 40px;
}

.docker-images-cleanup__header-icon :deep(.t-icon) {
  font-size: var(--td-font-size-title-large);
}

.docker-images-cleanup-summary {
  background: var(--td-brand-color-1);
  border: 1px solid var(--td-brand-color-3);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
}

.docker-images-cleanup-summary__stats {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.docker-images-cleanup-summary__stats > div {
  min-width: 0;
}

.docker-images-cleanup-summary__label {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
  margin-bottom: var(--graft-density-gap-4);
}

.docker-images-cleanup-summary__stats strong {
  color: var(--td-brand-color-7);
  font: var(--td-font-title-large);
  margin-right: var(--graft-density-gap-4);
}

.docker-images-cleanup-warning {
  margin: 0;
}

.docker-images-cleanup-preview {
  min-width: 0;
}

.docker-images-cleanup-section-head,
.docker-images-cleanup-footer {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.docker-images-cleanup-section-head h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.docker-images-cleanup-section-head span,
.docker-images-cleanup-footer span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-cleanup-preview-body {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(0, 1fr) 32px;
}

.docker-images-cleanup-table {
  min-width: 0;
}

.docker-images-cleanup-image {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: 18px minmax(0, 1fr);
  min-width: 0;
  width: 100%;
}

.docker-images-cleanup-image__icon {
  color: var(--td-text-color-secondary);
}

.docker-images-cleanup-image__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-images-cleanup-pager {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  justify-content: center;
}

.docker-images-cleanup-pager :deep(.t-button) {
  color: var(--td-text-color-secondary);
}

.docker-images-cleanup-pager :deep(.t-button:not(.t-is-disabled):hover) {
  color: var(--td-brand-color-7);
}

.docker-images-cleanup-footer strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-medium);
  margin-top: var(--graft-density-gap-4);
}

:deep(.docker-images-cleanup-dialog .t-dialog__body) {
  overflow: hidden;
}

:deep(.docker-images-cleanup-dialog .t-dialog__footer) {
  padding-top: 0;
}

.docker-images-pull-log {
  margin-top: var(--graft-density-gap-16);
}

@media (width <= 768px) {
  .docker-images-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .docker-images-cleanup-summary__stats {
    grid-template-columns: 1fr;
  }

  .docker-images-cleanup-section-head,
  .docker-images-cleanup-footer {
    align-items: flex-start;
    flex-direction: column;
  }

  .docker-images-cleanup-footer :deep(.t-space) {
    width: 100%;
  }

  .docker-images-cleanup-footer :deep(.t-button) {
    flex: 1;
  }
}
</style>
