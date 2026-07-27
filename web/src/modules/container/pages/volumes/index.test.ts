import { describe, expect, it } from 'vitest';

import detailContentText from '../../components/VolumeDetailContent.vue?raw';
import removalContentText from '../../components/VolumeRemovalConfirmContent.vue?raw';
import presentationText from '../../shared/volume-presentation.ts?raw';
import removalText from '../../shared/volume-removal.ts?raw';
import sourceText from './index.vue?raw';

describe('docker volume asset management page', () => {
  it('renders a compact mobile asset card instead of stacking every table field', () => {
    expect(sourceText).toContain('cards-visible');
    expect(sourceText).toContain('density-scope="viewport"');
    expect(sourceText).toContain('entity-card-layout="compact"');
    expect(sourceText).toContain('preserve-inactive');
    expect(sourceText).toContain('class="docker-volume-page__card"');
    expect(sourceText).toContain('class="docker-volume-page__card-primary"');
    expect(sourceText).toContain('class="docker-volume-page__card-secondary"');
    expect(sourceText).toContain('block-size: 9.25rem;');
    expect(sourceText).toContain('middleEllipsis(row.name, 31)');
  });

  it('navigates mobile cards directly to the renderable detail child route', () => {
    expect(sourceText).toContain('@click="openDetailPage(row)"');
    expect(sourceText).toContain('CONTAINER_BOOTSTRAP_ROUTE.VOLUME_DETAIL.pageRouteName');
    expect(sourceText).not.toContain('CONTAINER_BOOTSTRAP_ROUTE.VOLUME_DETAIL.routeName, params');
  });

  it('keeps the requested desktop column order and sortable capacity', () => {
    const selection = sourceText.indexOf("colKey: 'row-select'");
    const name = sourceText.indexOf("colKey: 'name'");
    const status = sourceText.indexOf("colKey: 'status'");
    const size = sourceText.indexOf("colKey: 'size'");
    const references = sourceText.indexOf("colKey: 'references'");
    const driver = sourceText.indexOf("colKey: 'driver'");
    const createdAt = sourceText.indexOf("colKey: 'created_at'");
    const actions = sourceText.indexOf("colKey: 'actions'");

    expect([selection, name, status, size, references, driver, createdAt, actions]).toEqual(
      [...[selection, name, status, size, references, driver, createdAt, actions]].sort((left, right) => left - right),
    );
    expect(sourceText).toContain("colKey: 'size',");
    expect(sourceText).toContain('sorter: true');
    expect(sourceText).not.toContain("colKey: 'context'");
  });

  it('defaults the server query to capacity descending and supports sort changes', () => {
    expect(sourceText).toContain("const sort = ref<TableSort>({ sortBy: 'size', descending: true })");
    expect(sourceText).toContain("sort_by: 'size_bytes'");
    expect(sourceText).toContain("? 'desc' : 'asc'");
    expect(sourceText).toContain('@sort-change="handleSortChange"');
  });

  it('uses the volume-specific three-state presentation', () => {
    expect(presentationText).toContain("status === 'used'");
    expect(presentationText).toContain("status === 'unused'");
    expect(presentationText).toContain("theme: 'success'");
    expect(presentationText).toContain("theme: 'warning'");
    expect(presentationText).toContain("theme: 'danger'");
    expect(sourceText).not.toContain("t('container.volume.metrics.referenceUnknown'");
    expect(sourceText).not.toContain("t('container.volume.unavailable'");
  });

  it('restores filtered compact-card empty states through the existing filter reset action', () => {
    expect(sourceText).toContain(":title=\"hasActiveFilters ? t('container.volume.pagination.empty')");
    expect(sourceText).toContain("hasActiveFilters ? t('container.volume.filters.reset')");
    expect(sourceText).toContain('<template v-if="hasActiveFilters" #action>');
    expect(sourceText).toContain('@click="resetFilters"');
  });

  it('formats compact-card dates with the locale-safe date-only helper', () => {
    expect(sourceText).toContain('formatLocaleDateOnly(value, locale)');
    expect(sourceText).not.toContain("formatLocaleDateTime(value, locale).split(' ')");
  });

  it('uses a compact drawer with shared detail content on desktop', () => {
    expect(sourceText).toContain('<resource-detail-layout');
    expect(sourceText).toContain('size="compact"');
    expect(sourceText).toContain('<volume-detail-content');
    expect(sourceText).toContain('surface="drawer"');
    expect(detailContentText).toContain("surface === 'page'");
    expect(detailContentText).toContain("t('container.resourceContext.configuration')");
    expect(detailContentText).not.toContain('<docker-resource-context-card');
  });

  it('shows relationships, storage facts, and the safe-cleanup empty state on the page surface', () => {
    expect(detailContentText).toContain('volume.container_references.length');
    expect(detailContentText).toContain("t('container.volume.detail.noContainers')");
    expect(detailContentText).toContain('safeCleanupCandidate');
    expect(detailContentText).toContain("t('container.volume.detail.actualUsage')");
  });

  it('routes every removal entry through the shared danger confirmation facts', () => {
    expect(sourceText).toContain('openVolumeRemovalConfirmation');
    expect(sourceText).toContain('confirmCleanupRemoval');
    expect(removalText).toContain('DialogPlugin.confirm');
    expect(removalText).toContain("theme: 'danger'");
    expect(removalContentText).toContain("t('container.volume.columns.name')");
    expect(removalContentText).toContain("t('container.volume.columns.size')");
    expect(removalContentText).toContain("t('container.volume.columns.mountedContainers')");
    expect(removalContentText).toContain("t('container.volume.removal.risk')");
  });

  it('preserves selection, cleanup, permissions, and container navigation', () => {
    expect(sourceText).toContain(':selected-row-keys="selectedRowKeys"');
    expect(sourceText).toContain('batchRemoveDockerVolumes');
    expect(sourceText).toContain('await cleanup.open();');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.VOLUME_REMOVE');
    expect(sourceText).toContain("query: { tab: 'storage' }");
  });
});
