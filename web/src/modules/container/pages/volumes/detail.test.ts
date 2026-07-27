import { describe, expect, it } from 'vitest';

import sourceText from './detail.vue?raw';

describe('docker volume detail page', () => {
  it('is a route page and returns through the renderable list child route', () => {
    expect(sourceText).toContain('presentation="page"');
    expect(sourceText).toContain('<volume-detail-content');
    expect(sourceText).toContain('surface="page"');
    expect(sourceText).toContain('CONTAINER_BOOTSTRAP_ROUTE.VOLUMES.pageRouteName');
    expect(sourceText).not.toContain('CONTAINER_BOOTSTRAP_ROUTE.VOLUMES.routeName });');
  });

  it('reuses the shared removal confirmation and returns only after successful deletion', () => {
    expect(sourceText).toContain('openVolumeRemovalConfirmation');
    expect(sourceText).toContain('confirmationName: candidate.name');
    expect(sourceText).toContain("forceRequired: candidate.relationship_status !== 'unused'");
    expect(sourceText).toContain('returnToList();');
  });
});
