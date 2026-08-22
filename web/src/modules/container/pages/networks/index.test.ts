import { describe, expect, it } from 'vitest';

import referenceListText from '../../shared/ContainerReferenceList.vue?raw';
import sourceText from './index.vue?raw';

describe('Docker network management page', () => {
  it('provides list-form-detail controls with confirmed destructive removal', () => {
    expect(sourceText).toContain('data-page-type="list-form-detail"');
    expect(sourceText).toContain('useDockerNetworkListQuery(networkListQuery)');
    expect(sourceText).toContain('useDockerNetworkDetailQuery(selectedNetworkId)');
    expect(sourceText).toContain('confirm_network_name: removeConfirmation.value');
    expect(sourceText).toContain('invalidateDockerNetworkQueries()');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.NETWORK_CREATE');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.NETWORK_REMOVE');
    expect(sourceText).toContain('@click="openRemoveDialog(row)"');
    expect(sourceText).toContain("t('container.networks.remove')");
    expect(sourceText).not.toContain('<table-action-menu');
  });

  it('uses TDesign drawers and dialogs instead of native browser dialogs', () => {
    expect(sourceText).toContain('<t-drawer');
    expect(sourceText).toContain('<t-dialog');
    expect(sourceText).not.toContain('window.confirm');
    expect(sourceText).not.toContain('window.alert');
  });

  it('submits permission-gated per-network Tasks without interpreting daemon partial results', () => {
    expect(sourceText).toContain(':selected-row-keys="isCompactDensity ? [] : selectedNetworkIds"');
    expect(sourceText).toContain('<management-batch-bar');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.NETWORK_REMOVE');
    expect(sourceText).toContain('Promise.allSettled');
    expect(sourceText).toContain('selectedNetworkNamesByID');
    expect(sourceText).toContain('confirm_network_name: selectedNetworkNamesByID.value[id]');
    expect(sourceText).not.toContain('container.networks.batch.removePartial');
    expect(sourceText).toContain('invalidateDockerNetworkQueries()');
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('limit: pagination.pageSize');
    expect(sourceText).toContain('usage:');
    expect(sourceText).toContain('size="small" theme="danger" variant="outline" @click="openBatchRemoveDialog"');
  });

  it('switches the shared paged-table slot to mobile resource cards without a horizontal table', () => {
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('v-model:current="pagination.current"');
    expect(sourceText).toContain('v-model:page-size="pagination.pageSize"');
    expect(sourceText).toContain(':cards-visible="true"');
    expect(sourceText).toContain('density-scope="viewport"');
    expect(sourceText).toContain('<responsive-card-list');
    expect(sourceText).toContain('docker-network-page__mobile-card');
    expect(sourceText).toContain('docker-network-page__mobile-card-name');
    expect(sourceText).toContain('overflow-wrap: anywhere;');
    expect(sourceText).toContain('networkQuery.isFetching.value');
  });

  it('uses responsive breakpoints to reduce tablet columns and move mobile actions into a consistent card action bar', () => {
    expect(sourceText).toContain('useViewportResponsiveVariant()');
    expect(sourceText).toContain("comfortable: ['name', 'context', 'status', 'operation']");
    expect(sourceText).toContain("resource: 'docker-network.list'");
    expect(sourceText).toContain('#toolbar-actions');
    expect(sourceText).toContain('<saved-query-view-control :controller="savedViews" />');
    expect(sourceText).toContain('networkActionOptions');
    expect(sourceText).toContain('<docker-resource-card-actions');
    expect(sourceText).toContain("danger: true, label: t('container.networks.remove')");
    expect(sourceText).toContain("t('container.networks.removeRisk')");
  });

  it('uses ResourceQueryPanel as the only list filter surface', () => {
    expect(sourceText).toContain('<resource-query-panel');
    expect(sourceText).toContain(':config="queryConfig"');
    expect(sourceText).toContain('v-model="resourceQueryState"');
    expect(sourceText).toContain('filterBuilder: { enabled: true }');
    expect(sourceText).toContain('filters: [');
    expect(sourceText).toContain('getDockerResourceSourceLabel(t, value as DockerResourceSource)');
    expect(sourceText).not.toContain('<management-toolbar');
    expect(sourceText).not.toContain('<docker-resource-context-filters');
    expect(sourceText).not.toContain('advancedFiltersVisible');
    expect(sourceText).not.toContain('advancedFiltersDrawerVisible');
    expect(sourceText).not.toContain('simple-filters');
    expect(sourceText).not.toContain('toolbar-after-search');
  });

  it('maps shared query state back to the network list query', () => {
    expect(sourceText).toContain('const resourceQueryState = computed<ResourceQueryState>');
    expect(sourceText).toContain('filters: { ...draftFilters }');
    expect(sourceText).toContain('resourceQueryState.value = value');
    expect(sourceText).toContain('applyFilters();');
    expect(sourceText).toContain('source: appliedFilters.value.source || undefined');
    expect(sourceText).toContain('compose_project: appliedFilters.value.compose_project || undefined');
  });

  it('shows localized feedback when saved-view operations fail', () => {
    expect(sourceText).toContain(
      "MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.networks.loadFailed')))",
    );
    expect(sourceText).not.toContain('`${operation}: ${String(error)}`');
  });

  it('uses the shared cleanup snapshot for removable unused networks', () => {
    expect(sourceText).toContain('useDockerCleanup<DockerNetwork>');
    expect(sourceText).toContain(':empty="!cleanup.loading.value && !cleanup.items.value.length"');
    expect(sourceText).toContain('<docker-cleanup-loading-host');
    expect(sourceText).toContain("usage: 'unused'");
    expect(sourceText).toContain('network.removable !== false');
    expect(sourceText).toContain('selectedNetworkIds');
    expect(sourceText).toContain('confirm_network_name: network.name');
    expect(sourceText).toContain('Promise.allSettled');
    expect(sourceText).toContain('await invalidateDockerNetworkQueries();');
  });

  it('keeps the list focused on source, relations, and relationship status', () => {
    expect(sourceText).toContain("colKey: 'context'");
    expect(sourceText).toContain("title: t('container.resourceContext.source')");
    expect(sourceText).toContain("colKey: 'containers'");
    expect(sourceText).toContain("colKey: 'status'");
    expect(sourceText).toContain('relationshipPresentation(row.relationship_status)');
    expect(sourceText).toContain('sourceDescription(row)');
    expect(sourceText).not.toContain(
      't-tag size="small" variant="light-outline">{{ sourceLabel(row.context.source) }}</t-tag>',
    );
    expect(sourceText).toContain('<container-reference-list');
    expect(referenceListText).toContain('references.slice(0, 2)');
    expect(referenceListText).toContain('references.slice(2)');
    expect(referenceListText).toContain('<t-popup');
    expect(referenceListText).toContain('trigger="hover"');
    expect(referenceListText).toContain('container-reference-list__badge');
    expect(sourceText).toContain('openContainerReference(reference.id)');
    expect(sourceText).not.toContain("colKey: 'labels'");
  });

  it('uses Context before relations and keeps metadata and removal in the drawer tail', () => {
    expect(sourceText).toContain('<docker-resource-context-card');
    expect(sourceText).toContain("t('container.resourceContext.relations')");
    expect(sourceText).toContain('<t-collapse');
    expect(sourceText).toContain('<container-danger-zone');
    expect(sourceText).toContain(':description="t(\'container.networks.removeRisk\')"');
    expect(sourceText).toContain("resource: 'docker-network.list'");
  });

  it('keeps network detail loading, error, empty, and data states mutually exclusive', () => {
    expect(sourceText).toContain('detailQuery.isError.value');
    expect(sourceText).toContain('v-else-if="detailQuery.data.value"');
    expect(sourceText).toContain('v-else-if="!detailQuery.isFetching.value"');
    expect(sourceText).toContain('container.networks.detailEmptyTitle');
    expect(sourceText).toContain('container.networks.detailLoadFailed');
    expect(sourceText).toContain('row.container_references?.length');
    expect(sourceText).toContain('detailQuery.data.value.container_references?.length');
  });
});
