import { describe, expect, it } from 'vitest';

import sourceText from './index.vue?raw';

describe('Docker network management page', () => {
  it('provides list-form-detail controls with confirmed destructive removal', () => {
    expect(sourceText).toContain('data-page-type="list-form-detail"');
    expect(sourceText).toContain('useDockerNetworkListQuery()');
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
    expect(sourceText).toContain('v-model:selected-row-keys="selectedNetworkIds"');
    expect(sourceText).toContain('<management-batch-bar');
    expect(sourceText).toContain('CONTAINER_PERMISSION_CODE.NETWORK_REMOVE');
    expect(sourceText).toContain('Promise.allSettled');
    expect(sourceText).toContain('container.networks.batch.removePartial');
    expect(sourceText).toContain('invalidateDockerNetworkQueries()');
    expect(sourceText).toContain(':pagination="paginationConfig"');
  });
});
