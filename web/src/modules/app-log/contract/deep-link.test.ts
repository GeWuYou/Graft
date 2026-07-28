import { describe, expect, it } from 'vitest';

import { buildAppLogLocation, parseAppLogRouteQuery } from './deep-link';

describe('app-log deep link', () => {
  it('preserves the quick preset as page context without accepting unrelated query fields', () => {
    expect(buildAppLogLocation({ quick_preset: 'warnings', severity: 'warn' })).toEqual({
      path: '/observability/application-logs',
      query: { quick_preset: 'warnings', severity: 'warn' },
    });

    expect(parseAppLogRouteQuery({ quick_preset: 'warnings', severity: 'warn', unsupported: 'value' })).toMatchObject({
      quick_preset: 'warnings',
      severity: 'warn',
    });
    expect(parseAppLogRouteQuery({ quick_preset: 'warnings', unsupported: 'value' })).not.toHaveProperty('unsupported');
  });
});
