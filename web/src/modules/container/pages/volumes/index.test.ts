import { describe, expect, it } from 'vitest';

import sourceText from './index.vue?raw';

describe('docker volume list page', () => {
  it('keeps detail content in a list drawer and removes the route-driven detail flow', () => {
    expect(sourceText).toContain('<t-drawer');
    expect(sourceText).toContain('getDockerVolume');
    expect(sourceText).not.toContain('useRouter');
  });

  it('keeps long names bounded and exposes the complete name through a tooltip', () => {
    expect(sourceText).toContain('table-layout="fixed"');
    expect(sourceText).toContain('middleEllipsis(row.name)');
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

  it('renders a filter-aware TDesign empty state without replacing table slots', () => {
    expect(sourceText).toContain('<template #empty>');
    expect(sourceText).toContain('<t-empty');
    expect(sourceText).toContain('hasActiveFilters');
    expect(sourceText).toContain('@click="resetFilters"');
    expect(sourceText).toContain('<template #name="{ row }">');
    expect(sourceText).toContain('<template #usage="{ row }"');
    expect(sourceText).toContain('<template #actions="{ row }">');
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
