import { describe, expect, it } from 'vitest';

import { actionTitle, resourceLabel } from './presentation';

function t(key: string) {
  if (key === 'audit.common.unknownResource') {
    return 'Unknown resource';
  }

  return key;
}

describe('audit presentation helpers', () => {
  it('uses request path before the empty secondary fallback for resource labels', () => {
    expect(resourceLabel({ request_path: '/api/auth/login' }, t)).toBe('/api/auth/login');
  });

  it('resolves known action keys before falling back to source labels', () => {
    expect(
      actionTitle(
        {
          action: 'ops.container.action.batch.stop',
          source: 'DOMAIN_EVENT',
        },
        (key: string) => {
          if (key === 'audit.actionLabel.ops.container.action.batch.stop') {
            return 'Batch Stop Containers';
          }
          if (key === 'audit.common.source.DOMAIN_EVENT') {
            return 'Domain Audit';
          }
          return key;
        },
      ),
    ).toBe('Batch Stop Containers');
  });

  it('localizes a security event resource when it repeats the event action', () => {
    expect(
      resourceLabel(
        {
          action: 'auth.token.invalid',
          resource_name: 'auth.token.invalid',
          source: 'SECURITY_EVENT',
        },
        (key: string) => {
          if (key === 'audit.actionLabel.auth.token.invalid') {
            return 'Invalid Access Token';
          }
          return key;
        },
      ),
    ).toBe('Invalid Access Token');
  });
});
