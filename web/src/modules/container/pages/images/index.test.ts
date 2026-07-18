import { describe, expect, it } from 'vitest';

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
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).toContain(':total="total"');
    expect(sourceText).toContain('summary.value.size_bytes');
    expect(sourceText).not.toContain('filteredImages');
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
    expect(sourceText).toContain("error_code: 'client_request_failed'");
    expect(sourceText).toContain('let hasUnknownResponse = false;');
    expect(sourceText).toContain('return { hasUnknownResponse, items: results };');
  });

  it('reloads cleanup candidates after an unknown chunk response without retrying deletion', () => {
    expect(sourceText).toContain('if (hasUnknownResponse) await reconcileCleanupCandidates(successfulIds);');
    expect(sourceText).toContain('const candidates = await fetchCleanupCandidates();');
    expect(sourceText).toContain('candidateIds.has(id) && !confirmedSuccessfulIds.has(id)');
    expect(sourceText).toContain('if (!cleanup || !hasUnknownResponse) cleanupDialogVisible.value = false;');
    expect(sourceText).toContain("MessagePlugin.error(t('container.images.cleanup.loadFailed'))");
  });

  it('keeps translated table columns reactive instead of unwrapping them during setup', () => {
    expect(sourceText).toContain("const columns = computed<TableProps['columns']>");
    expect(sourceText).toContain("const cleanupColumns = computed<TableProps['columns']>");
    expect(sourceText).not.toContain(']).value;');
  });

  it('disables remove confirmation until referenced images are explicitly forced', () => {
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
  });

  it('loads all unused images and selects them by default in the cleanup dialog', () => {
    expect(sourceText).toContain('unused: true');
    expect(sourceText).toContain('cleanupSelectedIds.value = all.map((image) => image.id);');
    expect(sourceText).toContain('<t-table');
    expect(sourceText).toContain(':selected-row-keys="cleanupSelectedIds"');
    expect(sourceText).toContain('@select-change="handleCleanupSelectChange"');
    expect(sourceText).toContain('cleanupPreviewImages');
    expect(sourceText).toContain('cleanupPreviewPage');
    expect(sourceText).toContain('cleanupPreviewPageCount');
    expect(sourceText).toContain('cleanupPreviewLimit = 8');
    expect(sourceText).toContain('cleanupSelectedSize');
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
    expect(sourceText).toContain('const preserved = cleanupSelectedIds.value.filter');
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
  });
});
