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
    expect(sourceText).not.toContain(':pagination="');
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
});
