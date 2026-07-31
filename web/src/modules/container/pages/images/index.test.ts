import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import taskTypesSourceText from '../../contract/task-types.ts?raw';
import cleanupSourceText from '../../shared/cleanup/use-docker-cleanup.ts?raw';
import sourceText from './index.vue?raw';

const permissionStoreMock = vi.hoisted(() => ({ hasPermission: vi.fn(() => true) }));

vi.mock('@/store', () => ({ usePermissionStore: () => permissionStoreMock }));
vi.mock('@/utils/logger', () => ({ createLogger: () => ({ error: vi.fn(), info: vi.fn(), warn: vi.fn() }) }));
vi.mock('@/utils/request', () => ({ isApiRequestError: () => false }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }) }));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: { error: vi.fn(), success: vi.fn() } }));
vi.mock('../../api/container', () => ({ getDockerImage: vi.fn(), getDockerImages: vi.fn() }));
vi.mock('../../api/image-actions', () => ({
  batchRemoveDockerImages: vi.fn(),
  pullDockerImage: vi.fn(),
  removeDockerImage: vi.fn(),
  tagDockerImage: vi.fn(),
}));
vi.mock('../../../task/contract/task-observer', () => ({
  isTerminalTaskStatus: () => false,
  observeTask: vi.fn(),
}));
vi.mock('../../../task/contract/task-ui', () => ({ TaskDetailDrawer: defineComponent({ name: 'TaskDetailDrawer' }) }));
vi.mock('../../shared/cleanup/use-docker-cleanup', () => ({
  useDockerCleanup: () => ({
    close: vi.fn(),
    error: { value: '' },
    execute: vi.fn(),
    fetch: vi.fn(),
    items: { value: [] },
    loading: { value: false },
    previewPage: { value: 1 },
    selectedIds: { value: [] },
    visible: { value: false },
  }),
}));
vi.mock('../../shared/docker-image-queries', () => ({
  useDockerImageQuery: () => ({
    data: {
      value: {
        items: [
          {
            container_references: [],
            created_at: '2026-07-01T00:00:00Z',
            id: 'sha256:image-1',
            labels: {},
            repository_digests: [],
            repo_tags: ['graft/web:latest'],
            size_bytes: 1024,
          },
        ],
        summary: { dangling: 0, in_use: 0, size_bytes: 1024, total: 1 },
        total: 1,
      },
    },
    error: { value: null },
    isError: { value: false },
    isFetching: { value: false },
    refetch: vi.fn(),
  }),
}));

import DockerImagesPage from './index.vue';

function mountSelectionPage() {
  return mount(DockerImagesPage, {
    global: {
      stubs: {
        'docker-resource-card-actions': defineComponent({
          emits: ['action'],
          setup:
            (_, { emit }) =>
            () =>
              h('button', {
                'data-testid': 'select-image-card',
                onClick: (event: MouseEvent) => {
                  event.stopPropagation();
                  emit('action', 'select');
                },
              }),
        }),
        'management-paged-table': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('section', slots.cards?.()),
        }),
        'resource-detail-layout': true,
        'tag-manager-drawer': true,
        'task-detail-drawer': true,
      },
    },
  });
}

describe('docker image list page', () => {
  it('centers a standalone detail loading indicator before the request resolves', () => {
    expect(sourceText).toContain('class="docker-images-detail-loading-host"');
    expect(sourceText).toContain('.docker-images-detail-loading-host {\n  min-height: 240px;');
    expect(sourceText).toContain('class="docker-images-detail-loading-host__indicator"');
    expect(sourceText).toContain('place-items: center;');
    expect(sourceText).toContain('docker-images-detail-loading-spin 1s linear infinite');
  });

  it('submits image pulls through the Task Runtime inside the container module page', () => {
    expect(sourceText).toContain('useDockerImageQuery');
    expect(sourceText).toContain('pullDockerImage');
    expect(sourceText).toContain('<task-detail-drawer');
    expect(sourceText).toContain('observeTask');
    expect(sourceText).not.toContain('LogBatchBuffer');
    expect(sourceText).not.toContain('LogRingBuffer');
    expect(sourceText).not.toContain('<log-viewer');
  });

  it('uses the shared server-paged table and summary contract', () => {
    expect(sourceText).toContain('<management-page-header\n      compact');
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).toContain(':total="total"');
    expect(sourceText).toContain('summary.value.size_bytes');
    expect(sourceText).not.toContain('filteredImages');
    expect(sourceText).toContain('<management-statistics-bar');
    expect(sourceText).toContain('<table-view-toolbar');
    expect(sourceText).not.toContain('docker-images-summary');
    expect(sourceText).toContain('summary.value.dangling');
    expect(sourceText).not.toContain('docker-images-metrics');
  });

  it('declares an entity presentation with compact cards and tablet column priorities', () => {
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).toContain('presentation="entity"');
    expect(sourceText).toContain('entity-card-layout="compact"');
    expect(sourceText).toContain("comfortable: ['tags', 'size', 'status', 'actions']");
    expect(sourceText).toContain(
      "spacious: ['row-select', 'tags', 'size', 'containers', 'status', 'created_at', 'actions']",
    );
  });

  it('uses detail actions instead of opening compact cards outside selection mode', () => {
    expect(sourceText).toContain('<template #cards>');
    expect(sourceText).toContain('class="docker-images-card"');
    expect(sourceText).toContain('formatBytes(image.size_bytes)');
    expect(sourceText).toContain("t('container.images.fields.containers')");
    expect(sourceText).toContain("t('container.images.fields.createdAt')");
    expect(sourceText).toContain('v-if="cardSelectionMode"');
    expect(sourceText).toContain("{ label: t('container.images.actions.select'), value: 'select' }");
    expect(sourceText).toContain('function handleCardClick(image: DockerImage)');
    expect(sourceText).toContain("'docker-images-card--selection-mode': cardSelectionMode");
    expect(sourceText).toContain(':role="cardSelectionMode ? \'button\' : undefined"');
    expect(sourceText).toContain(':tabindex="cardSelectionMode ? 0 : undefined"');
    expect(sourceText).toContain(':aria-pressed="cardSelectionMode ? isImageSelected(image) : undefined"');
    expect(sourceText).toContain('if (!cardSelectionMode.value) return;');
    expect(sourceText).toContain('function handleCardKeydown(event: KeyboardEvent, image: DockerImage)');
    expect(sourceText).toContain('event.target !== event.currentTarget');
    expect(sourceText).toContain('function setCardSelected(image: DockerImage, selected: boolean)');
    expect(sourceText).toContain('<docker-resource-card-actions');
    expect(sourceText).toContain('@detail="openDetail(image)"');
    expect(sourceText).toContain("danger: true, label: t('container.images.actions.remove')");
  });

  it('selects cards only in selection mode and exposes the selected state', async () => {
    const wrapper = mountSelectionPage();
    const card = wrapper.get('[data-testid="docker-image-card-sha256:image-1"]');

    expect(card.attributes('aria-pressed')).toBeUndefined();
    await card.trigger('click');
    expect(card.attributes('aria-pressed')).toBeUndefined();

    await wrapper.get('[data-testid="select-image-card"]').trigger('click');
    expect(card.attributes('aria-pressed')).toBe('true');

    await card.trigger('click');
    expect(card.attributes('aria-pressed')).toBe('false');

    await card.trigger('keydown.space');
    expect(card.attributes('aria-pressed')).toBe('true');

    await card.find('strong').trigger('keydown.space');
    expect(card.attributes('aria-pressed')).toBe('true');
  });

  it('places image removal in the shared container danger zone', () => {
    expect(sourceText).toContain('<container-danger-zone');
    expect(sourceText).toContain('v-if="selectedImage && canRemove"');
    expect(sourceText).toContain(':description="t(\'container.images.remove.risk\')"');
    expect(sourceText).toContain('@action="openRemove(selectedImage)"');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.IMAGE_REMOVE');
  });

  it('keeps the pull action primary and moves cleanup into compact overflow', () => {
    expect(sourceText).toContain('<template #compactActions>');
    expect(sourceText).toContain('compactHeaderActions');
    expect(sourceText).toContain('handleCompactHeaderAction');
    expect(sourceText).toContain('sticky-compact');
    expect(sourceText).toContain("t('container.images.searchCompact')");
    expect(sourceText).toContain('layout="chips"');
  });

  it('resets server pagination when submitting or clearing the keyword', () => {
    expect(sourceText).toContain('@enter="applyKeyword"');
    expect(sourceText).toContain('@clear="clearKeyword"');
    expect(sourceText).toContain('submittedKeyword.value = keyword.value.trim();');
    expect(sourceText).toContain("submittedKeyword.value = '';");
    expect(sourceText).toContain('pagination.current = 1;');
  });

  it('renders a TDesign empty state with a keyword reset action', () => {
    expect(sourceText).toContain('<template #empty>');
    expect(sourceText).toContain('<t-empty');
    expect(sourceText).toContain("t('container.images.clearFilter')");
    expect(sourceText).toContain('@click="clearKeyword"');
  });

  it('preserves registry ports and repository paths when deriving a tag target', () => {
    expect(sourceText).toContain("const lastSlash = reference.lastIndexOf('/');");
    expect(sourceText).toContain("const lastColon = reference.lastIndexOf(':');");
    expect(sourceText).toContain('lastColon > lastSlash ? reference.slice(0, lastColon) : reference');
  });

  it('opens the accepted pull Task and refreshes only after a successful terminal state', () => {
    expect(sourceText).toContain('createPullIdempotencyKey()');
    expect(sourceText).toContain('const crypto = globalThis.crypto;');
    expect(sourceText).toContain('crypto?.randomUUID?.()');
    expect(sourceText).toContain('crypto?.getRandomValues?.(entropy);');
    expect(sourceText).toContain('pullIdempotencySequence += 1;');
    expect(sourceText).toContain('container-image-pull-${Date.now()}-${pullIdempotencySequence}');
    expect(sourceText).toContain('pullTaskId.value = receipt.task_id;');
    expect(sourceText).toContain('pullTaskDrawerVisible.value = true;');
    expect(sourceText).toContain('observePullTask(receipt.task_id);');
    expect(sourceText).toContain('if (!isTerminalTaskStatus(task.status)) return;');
    expect(sourceText).toContain("if (task.status === 'success') void refresh();");
    expect(sourceText).toContain('stopPullTaskObserver();');
  });

  it('owns the Docker image pull Task type in the container contract', () => {
    expect(taskTypesSourceText).toContain("DOCKER_IMAGE_PULL: 'container.docker-image-pull.v1'");
    expect(sourceText).toContain("import { CONTAINER_TASK_TYPE } from '../../contract/task-types';");
    expect(sourceText).toContain('CONTAINER_TASK_TYPE.DOCKER_IMAGE_PULL');
    expect(sourceText).not.toContain("taskType === 'container.docker-image-pull.v1'");
  });

  it('supports cross-page selection and chunked batch removal', () => {
    expect(sourceText).toContain("{ colKey: 'row-select', type: 'multiple' as const, width: 48 }");
    expect(sourceText).toContain(':selected-row-keys="selectedRowKeys"');
    expect(sourceText).toContain('@select-change="handleSelectChange"');
    expect(sourceText).toContain('const preserved = selectedRowKeys.value.filter');
    expect(sourceText).toContain('index += 100');
    expect(sourceText).toContain('batchRemoveDockerImages');
    expect(sourceText).toContain('results.push(...response.items);');
    expect(sourceText).toContain('error_code: DOCKER_IMAGE_REMOVE_ERROR_CODES.UNKNOWN');
    expect(sourceText).toContain('unknownResponseIds.push(...chunkIds);');
    expect(sourceText).toContain('let requestError: unknown;');
    expect(sourceText).toContain('return { items: results, unknownResponseIds, requestError };');
  });

  it('renders every daemon-reported batch failure by stable error code without parsing raw Docker text', () => {
    expect(sourceText).toContain('DOCKER_IMAGE_REMOVE_ERROR_CODES');
    expect(sourceText).toContain('normalizeBatchFailureCode(item.error_code)');
    expect(sourceText).toContain('batchFailureGroups');
    expect(sourceText).toContain('batchResultDialogVisible.value = true;');
    expect(sourceText).toContain('showBatchFailureDetails(failed, items.length - failed.length);');
    expect(sourceText).not.toContain('items.slice(0, 5)');
    expect(sourceText).not.toContain('item.message ||');
  });

  it('keeps batch results visible and provides one tag-management entry for every multi-tag failure', () => {
    expect(sourceText).toContain('failure.code === DOCKER_IMAGE_REMOVE_ERROR_CODES.IMAGE_REFERENCED_BY_MULTIPLE_TAGS');
    expect(sourceText).toContain('@click="openBatchFailureTagManager(failure.id)"');
    expect(sourceText).toContain('function openBatchFailureTagManager(imageId: string)');
    expect(sourceText).toContain('restoreBatchResultAfterTagManager.value = true;');
    expect(sourceText).toContain('@update:visible="handleTagManagerVisibleChange"');
    expect(sourceText).toContain('function handleTagManagerVisibleChange(visible: boolean)');
    expect(sourceText).toContain('batchResultDialogVisible.value = true;');
    expect(sourceText).toContain('batchResultDialogVisible.value = false;');
  });

  it('keeps tag-conflict handling normal while preserving explicit force for container references', () => {
    expect(sourceText).toContain('await submitBatchRemove(selectedRowKeys.value.map(String), forceRemove.value);');
    expect(sourceText).toContain("t('container.images.batch.riskMultipleTags')");
    expect(sourceText).toContain("t('container.images.batch.riskContainerReference')");
    expect(sourceText).toContain("t('container.images.batch.normalRemovalOnly')");
    expect(sourceText).toContain('selectedBatchReferences');
  });

  it('reloads cleanup candidates after an unknown chunk response without retrying deletion', () => {
    expect(sourceText).toContain('if (hasUnknownResponse) await reconcileCleanupCandidates(successfulIds);');
    expect(sourceText).toContain('await cleanup.reconcile(confirmedSuccessfulIds);');
    expect(cleanupSourceText).toContain('candidateIds.has(id) && !confirmedSuccessfulIds.has(id)');
    expect(sourceText).toContain('if (!requestError && !hasUnknownResponse');
    expect(sourceText).toContain('if (!cleanup) cleanupDialogVisible.value = false;');
    expect(sourceText).toContain("MessagePlugin.error(t('container.images.cleanup.loadFailed'))");
  });

  it('clears normal-batch selections only when an uncertain deletion is confirmed missing', () => {
    expect(sourceText).toContain('await reconcileSelectedImages(unknownResponseIds);');
    expect(sourceText).toContain('selectedImages.value.set(id, await getDockerImage(id));');
    expect(sourceText).toContain('isApiRequestError(error) && error.status === 404');
    expect(sourceText).toContain('forgetSelectedImages(removedIds);');
    expect(sourceText).toContain('if (requestError || hasUnknownResponse)');
    expect(sourceText).toContain('showUnknownBatchResult();');
  });

  it('keeps translated table columns reactive instead of unwrapping them during setup', () => {
    expect(sourceText).toContain("const columns = computed<TableProps['columns']>");
    expect(sourceText).toContain("const cleanupColumns = computed<TableProps['columns']>");
    expect(sourceText).not.toContain(']).value;');
  });

  it('requires explicit force for images referenced by containers', () => {
    expect(sourceText).toContain('const removeConfirmButton = computed');
    expect(sourceText).toContain('!forceRemove.value');
    expect(sourceText).toContain('selectedImage.value?.container_references?.length');
    expect(sourceText).toContain('selectedBatchReferences.value.length');
  });

  it('uses a compact row menu and presents container references as tags with id tooltips', () => {
    expect(sourceText).toContain('<table-action-menu');
    expect(sourceText).toContain('container.images.actions.more');
    expect(sourceText).toContain('row.container_references');
    expect(sourceText).toContain(':content="container.id"');
    expect(sourceText).toContain("t('container.images.unused')");
    expect(sourceText).toContain("t('container.images.fields.status')");
    expect(sourceText).toContain("t('container.images.status.used')");
    expect(sourceText).toContain("t('container.images.status.unused')");
    expect(sourceText).toContain("t('container.images.status.dangling')");
    expect(sourceText).toContain('if (image.container_references?.length)');
  });

  it('keeps Image-ID-backed rows recognizable by repository tags and opens one tag manager', () => {
    expect(sourceText).toContain("title: t('container.images.fields.name')");
    expect(sourceText).toContain("value: 'manage-tags'");
    expect(sourceText).toContain('<tag-manager-drawer');
    expect(sourceText).toContain('@refreshed="handleTagManagerRefreshed"');
    expect(sourceText).toContain("t('container.images.actions.manageTags')");
    expect(sourceText).toContain("imageReference(imageTags(row)[0] ?? '').repository");
    expect(sourceText).not.toContain("{ colKey: 'id'");
  });

  it('organizes the detail drawer and truncates metadata with full-value tooltips', () => {
    expect(sourceText).toContain("t('container.images.detail.overview')");
    expect(sourceText).toContain("t('container.images.detail.basicInfo')");
    expect(sourceText).toContain("t('container.images.detail.metadata')");
    expect(sourceText).toContain('middleEllipsis(selectedImage.id)');
    expect(sourceText).toContain('middleEllipsis((selectedImage.repository_digests ?? []).join');
    expect(sourceText).toContain('t-tooltip :content="selectedImage.id"');
    expect(sourceText).toContain(
      "middleEllipsis(imageReference(imageTags(selectedImage)[0] ?? '').repository || '-', 44)",
    );
    expect(sourceText).toContain('theme="info"');
    expect(sourceText).toContain('<template #footer>');
    expect(sourceText).toContain('@action="openRemove(selectedImage)"');
  });

  it('previews a multi-tag Image delete failure without changing the remove request semantics', () => {
    expect(sourceText).toContain('multipleTagsPreflight');
    expect(sourceText).toContain('error.messageKey === dockerImageReferencedByMultipleTagsMessageKey');
    expect(sourceText).toContain('isMultipleTagFailure(error)');
    expect(sourceText).toContain('openFailedImageTagManager');
    expect(sourceText).toContain('removeDockerImage(selectedImage.value.id, { force: forceRemove.value })');
  });

  it('keeps image detail loading, error, empty, and data states mutually exclusive', () => {
    expect(sourceText).toContain('detailError');
    expect(sourceText).toContain('v-else-if="selectedImage"');
    expect(sourceText).toContain('v-else-if="!detailLoading"');
    expect(sourceText).toContain('container.images.detail.emptyTitle');
    expect(sourceText).toContain("resolveLocalizedErrorMessage(t, error, t('container.images.detail.loadFailed'))");
    expect(sourceText).toContain('selectedImage.value = null;');
    expect(sourceText).toContain('selectedImage.repository_digests?.length');
    expect(sourceText).toContain('image.repository_tags?.filter(Boolean) ?? []');
  });

  it('loads all unused images and selects them by default in the cleanup dialog', () => {
    expect(sourceText).toContain('unused: true');
    expect(cleanupSourceText).toContain('selectedIds.value = items.value.map((item) => item.id);');
    expect(sourceText).toContain('<t-table');
    expect(sourceText).toContain(':selected-row-keys="cleanupSelectedIds"');
    expect(sourceText).toContain('@select-change="handleCleanupSelectChange"');
    expect(sourceText).toContain('cleanupPreviewImages');
    expect(sourceText).toContain('cleanupPreviewPage');
    expect(sourceText).toContain('cleanupPreviewPageCount');
    expect(cleanupSourceText).toContain('const pageSize = options.pageSize ?? 8;');
    expect(sourceText).toContain('const cleanupSelectedSize = cleanup.selectedSize;');
    expect(sourceText).not.toContain('cleanupPagination');
    expect(sourceText).not.toContain('cleanupVisibleImages');
    expect(sourceText).not.toContain('selectCleanupPage');
    expect(sourceText).not.toContain('cleanupPreviewExpanded');
    expect(sourceText).not.toContain('cleanupPreviewExpand');
    expect(sourceText).toContain("t('container.images.cleanup.warning')");
    expect(sourceText).toContain("t('container.images.cleanup.removeSelected'");
    expect(sourceText).toContain('dialog-class-name="docker-images-cleanup-dialog"');
    expect(sourceText).toContain('docker-images-cleanup graft-scrollbar');
    expect(sourceText).toContain(':empty="!cleanupLoading && !cleanupImages.length"');
    expect(sourceText).toContain('<docker-cleanup-loading-host');
    expect(sourceText).toContain('max-height: calc(70vh - 120px);');
    expect(sourceText).toContain('overflow: auto;');
    expect(sourceText).toContain('<template #footer>');
  });

  it('presents cleanup as a summary, preview, and confirm flow', () => {
    expect(sourceText).toContain('docker-images-cleanup-summary');
    expect(sourceText).toContain("t('container.images.cleanup.candidateCount')");
    expect(sourceText).toContain("t('container.images.cleanup.releaseSize')");
    expect(sourceText).toContain('formatBytes(cleanupTotalSize)');
    expect(sourceText).toContain("t('container.images.cleanup.source')");
    expect(sourceText).toContain("t('container.images.cleanup.candidateTitle'");
    expect(sourceText).toContain("t('container.images.cleanup.selectedCount'");
    expect(sourceText).toContain("t('container.images.cleanup.footerRelease')");
    expect(sourceText).toContain(':disabled="!cleanupSelectedIds.length"');
    expect(sourceText).toContain('previousCleanupPage');
    expect(sourceText).toContain('nextCleanupPage');
    expect(sourceText).toContain('<arrow-up-icon />');
    expect(sourceText).toContain('<arrow-down-icon />');
    expect(sourceText).toContain("t('container.images.cleanup.imageColumn')");
    expect(sourceText).toContain("t('container.images.cleanup.statusColumn')");
    expect(sourceText).toContain("t('container.images.cleanup.sizeColumn')");
    expect(sourceText).toContain("{ colKey: 'row-select', type: 'multiple' as const, width: 48 }");
    expect(cleanupSourceText).toContain('const preserved = selectedIds.value.filter');
    expect(sourceText).not.toContain('<t-checkbox-group v-model="cleanupSelectedIds"');
    expect(sourceText).not.toContain('docker-images-cleanup-pagination-head');
    expect(sourceText).not.toContain("t('container.images.cleanup.selectPage')");
    expect(sourceText).not.toContain("t('container.images.cleanup.invertPage')");
    expect(sourceText).not.toContain('management-table-pagination');
  });

  it('reports full, partial, and failed batch removal outcomes', () => {
    expect(sourceText).toContain("t('container.images.batch.success'");
    expect(sourceText).toContain("t('container.images.batch.partial'");
    expect(sourceText).toContain("t('container.images.batch.failed'");
    expect(sourceText).toContain('successfulIds');
    expect(sourceText).toContain("t('container.images.batch.requestFailed')");
    expect(sourceText).toContain('batchResultDialogVisible.value = true;');
    expect(sourceText).toContain('@confirm="batchResultDialogVisible = false"');
    expect(sourceText).toContain('closeBatchDialogs();');
    expect(sourceText).toContain('removeDialogVisible.value = false;');
    expect(sourceText).toContain('cleanupDialogVisible.value = false;');
    expect(sourceText).toContain('logBatchRequestError');
    expect(sourceText).toContain('showUnknownBatchResult();');
  });
});
