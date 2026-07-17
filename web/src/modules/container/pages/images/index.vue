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
        <t-input v-model="keyword" class="management-list-search" clearable :placeholder="t('container.images.search')">
          <template #prefix-icon><search-icon /></template>
        </t-input>
      </template>
    </management-toolbar>

    <t-alert v-if="query.isError.value" theme="error" :message="t('container.images.loadFailed')" />
    <t-table
      row-key="id"
      :columns="columns"
      :data="filteredImages"
      :loading="query.isFetching.value"
      :pagination="{ pageSize: 20, total: filteredImages.length }"
      table-layout="fixed"
    >
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
        <t-tag :theme="row.containers ? 'warning' : 'default'" size="small" variant="light-outline">{{
          t('container.images.containerCount', { count: row.containers })
        }}</t-tag>
      </template>
      <template #created_at="{ row }">{{ formatLocaleDateTime(row.created_at, locale) }}</template>
      <template #actions="{ row }">
        <t-space size="small">
          <t-button size="small" variant="text" @click="openDetail(row)">{{
            t('container.images.actions.detail')
          }}</t-button>
          <t-button size="small" variant="text" @click="openTag(row)">{{ t('container.images.actions.tag') }}</t-button>
          <t-button size="small" theme="danger" variant="text" @click="openRemove(row)">{{
            t('container.images.actions.remove')
          }}</t-button>
        </t-space>
      </template>
    </t-table>

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
      :confirm-btn="t('container.images.actions.remove')"
      :confirm-loading="removing"
      @confirm="submitRemove"
    >
      <p>{{ t('container.images.remove.confirm', { image: selectedImage ? shortId(selectedImage.id) : '' }) }}</p>
      <p v-if="selectedImage?.containers" class="docker-images-risk">
        {{ t('container.images.remove.inUse', { count: selectedImage.containers }) }}
      </p>
      <t-checkbox v-if="selectedImage?.containers" v-model="forceRemove">{{
        t('container.images.remove.force')
      }}</t-checkbox>
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
// 镜像页面只管理当前 Docker runtime 的镜像快照；拉取日志是短生命周期 UI 状态，不进入 Query 缓存。
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onUnmounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementPageHeader, ManagementToolbar } from '@/shared/components/management';
import {
  formatBytes,
  formatLocaleDateTime,
  LogBatchBuffer,
  LogRingBuffer,
  LogViewer,
  type StructuredLogEntry,
} from '@/shared/observability';

import { type DockerImageRecord, getDockerImage } from '../../api/container';
import { type DockerImagePullEvent, pullDockerImage, removeDockerImage, tagDockerImage } from '../../api/image-actions';
import { useDockerImageQuery } from '../../shared/docker-image-queries';

type DockerImage = DockerImageRecord;

const { locale, t } = useI18n();
const query = useDockerImageQuery();
const keyword = ref('');
const selectedImage = ref<DockerImage | null>(null);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const tagDialogVisible = ref(false);
const removeDialogVisible = ref(false);
const pullDrawerVisible = ref(false);
const tagging = ref(false);
const removing = ref(false);
const forceRemove = ref(false);
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
const filteredImages = computed(() => {
  const normalized = keyword.value.trim().toLowerCase();
  if (!normalized) return images.value;
  return images.value.filter((image) =>
    [image.id, ...image.repository_tags, ...image.repository_digests].some((value) =>
      value.toLowerCase().includes(normalized),
    ),
  );
});
const metrics = computed(() => [
  { label: t('container.images.metrics.total'), value: images.value.length },
  {
    label: t('container.images.metrics.size'),
    value: formatBytes(images.value.reduce((total, image) => total + image.size_bytes, 0)),
  },
  { label: t('container.images.metrics.inUse'), value: images.value.filter((image) => image.containers > 0).length },
  {
    label: t('container.images.metrics.dangling'),
    value: images.value.filter((image) => image.repository_tags.length === 0).length,
  },
]);
const columns: TableProps['columns'] = computed(() => [
  { colKey: 'tags', title: t('container.images.fields.tags'), ellipsis: true, minWidth: 260 },
  { colKey: 'id', title: t('container.images.fields.id'), width: 150 },
  { colKey: 'size', title: t('container.images.fields.size'), width: 120 },
  { colKey: 'containers', title: t('container.images.fields.containers'), width: 120 },
  { colKey: 'created_at', title: t('container.images.fields.createdAt'), width: 180 },
  { colKey: 'actions', title: t('container.images.fields.actions'), width: 220, fixed: 'right' as const },
]).value;

function imageTags(image: DockerImage) {
  return image.repository_tags.filter(Boolean);
}
function shortId(value: string) {
  return value.replace(/^sha256:/, '').slice(0, 12);
}
function refresh() {
  return query.refetch();
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
  tagForm.repository = imageTags(image)[0]?.split(':')[0] ?? '';
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
  if (!selectedImage.value || (selectedImage.value.containers > 0 && !forceRemove.value)) return;
  removing.value = true;
  try {
    await removeDockerImage(selectedImage.value.id, { force: forceRemove.value });
    removeDialogVisible.value = false;
    await refresh();
    MessagePlugin.success(t('container.images.remove.success'));
  } catch {
    MessagePlugin.error(t('container.images.remove.failed'));
  } finally {
    removing.value = false;
  }
}
async function startPull() {
  if (!pullReference.value.trim() || pulling.value) return;
  pullLogBuffer.clear();
  pullLogEntries.value = [];
  pullLogVersion.value = 0;
  pulling.value = true;
  pullController = new AbortController();
  try {
    await pullDockerImage({ reference: pullReference.value.trim() }, pullController.signal, appendPullEvent);
    pullLogBatcher.flush();
    await refresh();
    MessagePlugin.success(t('container.images.pull.success'));
  } catch (error) {
    if (!pullController?.signal.aborted) {
      appendPullEvent({ error: true, status: error instanceof Error ? error.message : String(error) });
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

.docker-images-pull-log {
  margin-top: var(--graft-density-gap-16);
}

@media (width <= 768px) {
  .docker-images-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
