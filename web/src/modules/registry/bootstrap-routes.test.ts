import { describe, expect, it } from 'vitest';

import { registryBootstrapRouteRegistrations } from './bootstrap-routes';

describe('registry bootstrap route registrations', () => {
  it('registers the registry list as a paged table surface', () => {
    expect(registryBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        meta: expect.objectContaining({
          pageKind: 'list',
          pageSurface: 'paged-table',
        }),
      }),
    );
  });
});
