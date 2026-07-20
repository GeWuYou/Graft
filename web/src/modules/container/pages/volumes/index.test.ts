import { describe, expect, it } from 'vitest';

import sourceText from './index.vue?raw';

describe('docker volume list page', () => {
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
    expect(sourceText).toContain("{ colKey: 'name', title: t('container.volume.columns.name'), width: 280 }");
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
    expect(sourceText).toContain('for (let offset = firstPage.items.length; offset < firstPage.total; offset += 100)');
    expect(sourceText).toContain('await cleanup.open();');
    expect(sourceText).toContain('cleanup.select');
    expect(sourceText).toContain('cleanup.totalSize.value');
    expect(sourceText).toContain('for (let index = 0; index < ids.length; index += 50)');
    expect(sourceText).toContain('batchRemoveDockerVolumes({ names: chunk, force: false })');
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
    expect(sourceText).toContain('row.container_references.slice(0, 2)');
    expect(sourceText).toContain('selectedVolume.container_references');
    expect(sourceText).toContain('openContainerReference(reference.id)');
    expect(sourceText).toContain("t('container.volume.detail.references')");
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
});
