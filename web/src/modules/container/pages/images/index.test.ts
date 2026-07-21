import { describe, expect, it } from 'vitest';

import cleanupSourceText from '../../shared/cleanup/use-docker-cleanup.ts?raw';
import sourceText from './index.vue?raw';

describe('docker image list page', () => {
  it('keeps image actions and pull logs inside the container module page', () => {
    expect(sourceText).toContain('useDockerImageQuery');
    expect(sourceText).toContain('pullDockerImage');
    expect(sourceText).toContain('LogBatchBuffer');
    expect(sourceText).toContain('LogRingBuffer');
    expect(sourceText).toContain('<log-viewer');
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

  it('keeps responsive data-table behavior in the shared paged-table boundary', () => {
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).not.toContain('presentation="entity"');
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

  it('requires a completed pull event and rejects error events before refresh or success', () => {
    expect(sourceText).toContain('if (event.error)');
    expect(sourceText).toContain("throw new Error(event.status || 'Docker image pull failed.')");
    expect(sourceText).toContain('if (!pullCompleted) throw new Error');
    expect(sourceText).toContain("MessagePlugin.success(t('container.images.pull.success'))");
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
    expect(sourceText).toContain('@click="openRemove(selectedImage)"');
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
