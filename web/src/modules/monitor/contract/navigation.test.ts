import { describe, expect, it } from 'vitest';

import { buildMonitorLocationFromOrigin, parseMonitorOriginQuery } from './navigation';

describe('monitor request-performance origin', () => {
  it('restores the request-performance page and selected trend range', () => {
    const origin = parseMonitorOriginQuery({
      monitorView: 'request-performance',
      monitorTrendRange: '30m',
    });

    expect(origin).toEqual({ view: 'request-performance', trendRange: '30m' });
    expect(buildMonitorLocationFromOrigin(origin!)).toEqual({
      path: '/observability/request-performance',
      query: {
        monitorView: 'request-performance',
        monitorTrendRange: '30m',
      },
    });
  });
});
