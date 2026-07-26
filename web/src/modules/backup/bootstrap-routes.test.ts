import { describe, expect, it } from 'vitest';

import { backupBootstrapRouteRegistrations } from './bootstrap-routes';

describe('backup bootstrap route registrations', () => {
  it('registers Backup history as the canonical Platform entry', () => {
    expect(backupBootstrapRouteRegistrations).toEqual([
      expect.objectContaining({
        menuPath: '/platform/backups',
        routeName: 'PlatformBackupList',
        meta: expect.objectContaining({ pageKind: 'list', pageSurface: 'form-detail', tabGroup: 'platform' }),
      }),
    ]);
  });
});
