// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import {
  buildAuditLogsLocationWithOrigin,
  buildAuditRelatedActorLocation,
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
} from './navigation';

describe('audit navigation context', () => {
  it('preserves monitor origin on audit locations', () => {
    expect(
      buildAuditLogsLocationWithOrigin(
        { request_id: 'req-1' },
        { view: 'overview', trendRange: '10m', anomalyKey: 'cpu_pressure', scopeRef: 'runtime:cpu' },
      ),
    ).toEqual({
      path: '/audit/logs',
      query: {
        request_id: 'req-1',
        monitorView: 'overview',
        monitorTrendRange: '10m',
        monitorAnomalyKey: 'cpu_pressure',
        monitorScopeRef: 'runtime:cpu',
      },
    });
  });

  it('restores canonical monitor return target from audit route query', () => {
    const query = {
      request_id: 'req-1',
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

  it('builds actor locations from visible actor filters only', () => {
    expect(
      buildAuditRelatedActorLocation('alice', 42, {
        view: 'overview',
        trendRange: '10m',
        anomalyKey: 'cpu_pressure',
        scopeRef: 'runtime:cpu',
      }),
    ).toEqual({
      path: '/audit/logs',
      query: {
        actor: 'alice',
        monitorView: 'overview',
        monitorTrendRange: '10m',
        monitorAnomalyKey: 'cpu_pressure',
        monitorScopeRef: 'runtime:cpu',
      },
    });
  });
});
