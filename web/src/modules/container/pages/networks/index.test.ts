import { describe, expect, it } from 'vitest';

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
  });

  it('uses TDesign drawers and dialogs instead of native browser dialogs', () => {
    expect(sourceText).toContain('<t-drawer');
    expect(sourceText).toContain('<t-dialog');
    expect(sourceText).not.toContain('window.confirm');
    expect(sourceText).not.toContain('window.alert');
  });

  it('supports permission-gated batch removal with pagination and partial results', () => {
    expect(sourceText).toContain(':selected-row-keys="selectedNetworkIds"');
    expect(sourceText).toContain('<management-batch-bar');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.NETWORK_REMOVE');
    expect(sourceText).toContain('Promise.allSettled');
    expect(sourceText).toContain('selectedNetworkNamesByID');
    expect(sourceText).toContain('confirm_network_name: selectedNetworkNamesByID.value[id]');
    expect(sourceText).toContain('container.networks.batch.removePartial');
    expect(sourceText).toContain('invalidateDockerNetworkQueries()');
    expect(sourceText).toContain('<management-paged-table');
    expect(sourceText).toContain('limit: pagination.pageSize');
    expect(sourceText).toContain('usage:');
  });

  it('uses the shared cleanup snapshot for removable unused networks', () => {
    expect(sourceText).toContain('useDockerCleanup<DockerNetwork>');
    expect(sourceText).toContain("usage: 'unused'");
    expect(sourceText).toContain('network.removable !== false');
    expect(sourceText).toContain('selectedNetworkIds');
    expect(sourceText).toContain('confirm_network_name: network.name');
    expect(sourceText).toContain('Promise.allSettled');
    expect(sourceText).toContain('await invalidateDockerNetworkQueries();');
  });

  it('keeps the list focused on context, relations, and relationship status', () => {
    expect(sourceText).toContain("colKey: 'context'");
    expect(sourceText).toContain("colKey: 'containers'");
    expect(sourceText).toContain("colKey: 'status'");
    expect(sourceText).toContain('relationshipPresentation(row.relationship_status)');
    expect(sourceText).toContain('row.context.compose_project');
    expect(sourceText).toContain('row.container_references.slice(0, 2)');
    expect(sourceText).toContain('openContainerReference(reference.id)');
    expect(sourceText).not.toContain("colKey: 'labels'");
  });

  it('uses Context before relations and keeps metadata and removal in the drawer tail', () => {
    expect(sourceText).toContain('<docker-resource-context-card');
    expect(sourceText).toContain("t('container.resourceContext.relations')");
    expect(sourceText).toContain('<t-collapse');
    expect(sourceText).toContain("t('container.resourceContext.dangerZone')");
    expect(sourceText).toContain('advancedFiltersVisible');
  });
});
