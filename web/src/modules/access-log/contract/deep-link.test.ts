import { describe, expect, it } from 'vitest';

import { buildAccessLogLocation, buildAccessLogRequestLocation, parseAccessLogRouteQuery } from './deep-link';

describe('access-log monitor origin', () => {
  it('preserves request-performance origin outside the backend filter contract', () => {
    expect(
      buildAccessLogLocation(
        {
          quick_preset: 'status5xx',
          status_code: '503',
          occurred_from: '2026-07-14T08:00:00Z',
          occurred_to: '2026-07-14T08:10:00Z',
        },
        { view: 'request-performance', trendRange: '10m' },
      ),
    ).toEqual({
      path: '/observability/access-logs',
      query: {
        quick_preset: 'status5xx',
        status_code: '503',
        occurred_from: '2026-07-14T08:00:00Z',
        occurred_to: '2026-07-14T08:10:00Z',
        monitorView: 'request-performance',
        monitorTrendRange: '10m',
      },
    });

    const parsed = parseAccessLogRouteQuery({
      status_code: '503',
      quick_preset: 'status5xx',
      monitorView: 'request-performance',
      monitorTrendRange: '10m',
    });
    expect(parsed).toMatchObject({ quick_preset: 'status5xx', status_code: '503' });
    expect(parsed).not.toHaveProperty('monitorView');
    expect(parsed).not.toHaveProperty('monitorTrendRange');
  });

  it('adds origin context to request-id deep links', () => {
    expect(buildAccessLogRequestLocation('request-1', { view: 'request-performance', trendRange: '1h' })).toEqual({
      path: '/observability/access-logs',
      query: {
        request_id: 'request-1',
        monitorView: 'request-performance',
        monitorTrendRange: '1h',
      },
    });
  });
});
