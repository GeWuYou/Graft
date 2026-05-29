import { describe, expect, it } from 'vitest';

import {
  buildAuditLogsLocationWithOrigin,
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
} from './navigation';

describe('audit navigation context', () => {
  it('preserves monitor origin on audit locations', () => {
    expect(
      buildAuditLogsLocationWithOrigin(
        { requestId: 'req-1' },
        { view: 'overview', trendRange: '10m', anomalyKey: 'cpu_pressure', scopeRef: 'runtime:cpu' },
      ),
    ).toEqual({
      path: '/audit/logs',
      query: {
        requestId: 'req-1',
        monitorView: 'overview',
        monitorTrendRange: '10m',
        monitorAnomalyKey: 'cpu_pressure',
        monitorScopeRef: 'runtime:cpu',
      },
    });
  });

  it('restores canonical monitor return target from audit route query', () => {
    const query = {
      requestId: 'req-1',
      monitorView: 'dependencies',
      monitorTrendRange: '30m',
      monitorAnomalyKey: 'dependency_status_degraded',
      monitorScopeRef: 'postgresql',
    };

    expect(resolveAuditNavigationContext(query).monitorOrigin).toEqual({
      view: 'dependencies',
      trendRange: '30m',
      anomalyKey: 'dependency_status_degraded',
      scopeRef: 'postgresql',
    });

    expect(buildMonitorReturnLocation(query)).toEqual({
      path: '/server/dependencies',
      query: {
        monitorView: 'dependencies',
        monitorTrendRange: '30m',
        monitorAnomalyKey: 'dependency_status_degraded',
        monitorScopeRef: 'postgresql',
      },
    });
  });
});
