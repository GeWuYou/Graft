import { describe, expect, it } from 'vitest';

import { auditBootstrapRouteRegistrations } from './bootstrap-routes';
import { AUDIT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

describe('audit bootstrap route registrations', () => {
  it('uses the canonical audit bootstrap identity contract values', () => {
    expect(auditBootstrapRouteRegistrations).toHaveLength(3);
    expect(auditBootstrapRouteRegistrations).toEqual([
      expect.objectContaining({
        ...AUDIT_BOOTSTRAP_ROUTE.OVERVIEW,
        meta: expect.objectContaining({
          domain: 'audit',
          dashboard: true,
          pageKind: 'overview',
        }),
      }),
      expect.objectContaining({
        ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
        meta: expect.objectContaining({
          domain: 'audit',
          pageKind: 'list',
        }),
      }),
      expect.objectContaining({
        ...AUDIT_BOOTSTRAP_ROUTE.INCIDENT_DETAIL,
        meta: expect.objectContaining({
          domain: 'audit',
          pageKind: 'detail',
        }),
      }),
    ]);
  });
});
