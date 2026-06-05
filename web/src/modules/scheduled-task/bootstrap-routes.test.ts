import { describe, expect, it } from 'vitest';

import { scheduledTaskBootstrapRouteRegistrations } from './bootstrap-routes';
import { SCHEDULED_TASK_BOOTSTRAP_ROUTE } from './contract/bootstrap';

describe('scheduled task bootstrap route registrations', () => {
  it('uses the canonical scheduled task bootstrap identity contract values', () => {
    expect(scheduledTaskBootstrapRouteRegistrations).toHaveLength(1);
    expect(scheduledTaskBootstrapRouteRegistrations[0]).toMatchObject({
      ...SCHEDULED_TASK_BOOTSTRAP_ROUTE.LIST,
      meta: expect.objectContaining({
        tabGroup: 'server',
        pageKind: 'list',
      }),
    });
  });
});
