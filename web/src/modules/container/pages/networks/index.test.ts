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

  it('renders network attributes, labels, and localized creation time in the list', () => {
    expect(sourceText).toContain("'created_at'");
    expect(sourceText).toContain('formatLocaleDateTime(row.created_at, locale)');
    expect(sourceText).toContain("t('container.networks.noAttributes')");
    expect(sourceText).toContain('Object.entries(row.labels ?? {})');
    expect(sourceText).toContain('`${key}=${value}`');
    expect(sourceText).toContain('<t-tooltip');
  });

  it('keeps compact metadata columns while reserving room for labels', () => {
    expect(sourceText).toContain("{ colKey: 'name', title: t('container.networks.fields.name'), width: 360");
    expect(sourceText).toContain("{ colKey: 'driver', title: t('container.networks.fields.driver'), width: 120");
    expect(sourceText).toContain("{ colKey: 'scope', title: t('container.networks.fields.scope'), width: 120");
    expect(sourceText).toContain("{ colKey: 'labels', title: t('container.networks.labels'), width: 500 }");
    expect(sourceText).toContain("{ colKey: 'created_at', title: t('container.networks.fields.createdAt'), width: 180");
  });
});
