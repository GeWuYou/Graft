import { describe, expect, it } from 'vitest';

import referenceListText from '../../shared/ContainerReferenceList.vue?raw';
import sourceText from './index.vue?raw';

describe('docker volume list page', () => {
  it('centers a standalone detail loading indicator before the request resolves', () => {
    expect(sourceText).toContain('class="docker-volume-page__detail-loading-host"');
    expect(sourceText).toContain('.docker-volume-page__detail-loading-host {\n  min-height: 240px;');
    expect(sourceText).toContain('class="docker-volume-page__detail-loading-host__indicator"');
    expect(sourceText).toContain('place-items: center;');
    expect(sourceText).toContain('docker-volume-detail-loading-spin 1s linear infinite');
  });

  it('keeps volume detail content in a drawer while allowing reference navigation', () => {
    expect(sourceText).toContain('<t-drawer');
    expect(sourceText).toContain('getDockerVolume');
    expect(sourceText).toContain('useRouter');
    expect(sourceText).toContain("query: { tab: 'storage' }");
  });

  it('keeps long names bounded and exposes the complete name through a tooltip', () => {
    expect(sourceText).toContain('table-layout="fixed"');
    expect(sourceText).toContain('middleEllipsis(row.name, 31)');
    expect(sourceText).toContain(':content="row.name"');
    expect(sourceText).toContain("const columns = computed<TableProps['columns']>(() => [");
    expect(sourceText).toContain("{ colKey: 'name', title: t('container.volume.columns.name'), minWidth: 280 }");
  });

  it('makes volume identity, ownership, capacity, and relationship status scannable in one row', () => {
    expect(sourceText).toContain("t('container.volume.types.named')");
    expect(sourceText).toContain("t('container.volume.types.anonymous')");
    expect(sourceText).toContain('const anonymousVolumeName = /^[a-f0-9]{64}$/i;');
    expect(sourceText).toContain('function sourceDescription(row: VolumeRow)');
    expect(sourceText).toContain("colKey: 'size'");
    expect(sourceText).toContain("t('container.volume.columns.mountedContainers')");
    expect(sourceText).toContain("t('container.volume.columns.status')");
    expect(sourceText).toContain("title: t('container.resourceContext.source')");
    expect(sourceText).not.toContain("t('container.volume.columns.references')");
    expect(sourceText).not.toContain("t('container.volume.columns.usage')");
  });

  it('keeps mounted-container overflow actionable for pointer and keyboard users', () => {
    expect(sourceText).toContain('<container-reference-list');
    expect(referenceListText).toContain('<t-popup');
    expect(referenceListText).toContain('trigger="hover"');
    expect(referenceListText).toContain('@focus="overflowVisible = true"');
    expect(referenceListText).toContain('@keydown.esc.prevent="overflowVisible = false"');
    expect(referenceListText).toContain('@keydown.enter.prevent="emit(\'open\', reference.id)"');
    expect(referenceListText).toContain('@keydown.space.prevent="emit(\'open\', reference.id)"');
    expect(referenceListText).toContain('referenceTooltip(reference)');
  });

  it('provides selection and a batch removal request integration', () => {
    expect(sourceText).toContain("{ colKey: 'row-select', type: 'multiple' as const, width: 48 }");
    expect(sourceText).toContain(':selected-row-keys="selectedRowKeys"');
    expect(sourceText).toContain('@select-change="handleSelectChange"');
    expect(sourceText).toContain('function handleBatchRemove()');
    expect(sourceText).toContain('batchRemoveDockerVolumes');
  });

  it('cleans unused volumes with paged candidates and cross-page selection', () => {
    expect(sourceText).toContain("listDockerVolumes({ limit: 100, offset: 0, usage: 'unused' })");
    expect(sourceText).toContain('while (all.length < firstPage.total)');
    expect(sourceText).toContain('offset: all.length');
    expect(sourceText).toContain('if (!page.items.length) break;');
    expect(sourceText).toContain('await cleanup.open();');
    expect(sourceText).toContain('cleanup.select');
    expect(sourceText).toContain('cleanup.totalSize.value');
    expect(sourceText).toContain('for (let index = 0; index < ids.length; index += 50)');
    expect(sourceText).toContain('batchRemoveDockerVolumes({ names: chunk, force: false })');
    expect(sourceText).toContain('requestError = cause;');
    expect(sourceText).toContain('break;');
    expect(sourceText).toContain('cleanup.partial');
  });

  it('does not append CSS ellipsis to the already middle-ellipsized name', () => {
    expect(sourceText).toContain('middleEllipsis(row.name, 31)');
    expect(sourceText).toContain("{ colKey: 'name', title: t('container.volume.columns.name'), minWidth: 280 }");
    expect(sourceText).toContain('function middleEllipsis(value: string, maxLength = 31)');
    expect(sourceText).not.toContain("title: t('container.volume.columns.name'), ellipsis: true");
    expect(sourceText).not.toContain('text-overflow: ellipsis;');
  });

  it('uses the shared paged table with an explicit filter-aware empty state', () => {
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('<template #empty>');
    expect(sourceText).toContain('<t-empty');
    expect(sourceText).toContain('hasActiveFilters');
    expect(sourceText).toContain('@click="resetFilters"');
    expect(sourceText).toContain('<template #name="{ row }">');
    expect(sourceText).toContain('<template #references="{ row }">');
    expect(sourceText).toContain('<template #actions="{ row }">');
    expect(sourceText).toContain('<table-action-menu');
  });

  it('keeps the paged table on the shared data presentation for narrow containers', () => {
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).not.toContain('presentation="entity"');
  });

  it('uses the full runtime summary for four aligned statistics', () => {
    expect(sourceText).toContain('<management-statistics-bar');
    expect(sourceText).toContain("t('container.volume.metrics.total')");
    expect(sourceText).toContain("t('container.volume.metrics.inUse')");
    expect(sourceText).toContain("t('container.volume.metrics.unused')");
    expect(sourceText).toContain("t('container.volume.metrics.size')");
    expect(sourceText).toContain('volumeSummary.value?.size_bytes === null');
    expect(sourceText).toContain('response.summary');
  });

  it('shows sanitized container references in the table and detail drawer', () => {
    expect(sourceText).toContain('<container-reference-list');
    expect(referenceListText).toContain('references.slice(0, 2)');
    expect(sourceText).toContain('selectedVolume.container_references');
    expect(sourceText).toContain('openContainerReference(reference.id)');
    expect(sourceText).toContain("t('container.resourceContext.relations')");
    expect(sourceText).toContain("t('container.volume.metrics.referenceUnknown'");
  });

  it('uses TDesign controls in removal confirmations and avoids duplicate filter refreshes', () => {
    expect(sourceText).toContain('Input,');
    expect(sourceText).toContain('Checkbox,');
    expect(sourceText).not.toContain("h('input'");
    expect(sourceText).toContain('defaultValue: typedName');
    expect(sourceText).toContain('defaultChecked: isChecked()');
    expect(sourceText).toContain('const previousPage = pagination.current;');
    expect(sourceText).toContain('if (previousPage === 1) void refresh();');
  });

  it('keeps Context and relationship status in the list while moving technical filters behind More Filters', () => {
    expect(sourceText).toContain("colKey: 'context'");
    expect(sourceText).toContain("colKey: 'status'");
    expect(sourceText).toContain('advancedFiltersVisible');
    expect(sourceText).toContain('<docker-resource-context-card');
    expect(sourceText).toContain('<t-collapse');
    expect(sourceText).toContain("t('container.resourceContext.dangerZone')");
  });

  it('keeps volume detail loading, error, empty, and data states mutually exclusive', () => {
    expect(sourceText).toContain('v-else-if="selectedVolume"');
    expect(sourceText).toContain('v-else-if="!detailLoading"');
    expect(sourceText).toContain('container.volume.detail.emptyTitle');
    expect(sourceText).toContain('detailError');
    expect(sourceText).toContain('selectedVolume.value = null;');
  });

  it('consumes the canonical generated permission contract for dangerous actions', () => {
    expect(sourceText).toContain("from '@/contracts/generated/modules/container'");
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.VOLUME_REMOVE');
    expect(sourceText).not.toContain("from '../../contract/permissions'");
    expect(sourceText).not.toContain('ops.container.volume.remove');
    expect(sourceText).toContain('v-if="canRemove"');
    expect(sourceText).toContain('...(canRemove.value');
    expect(sourceText).toContain('if (!canRemove.value) return;');
  });
});
