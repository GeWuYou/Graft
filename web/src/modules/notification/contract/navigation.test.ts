// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { NOTIFICATION_NAVIGATION_KIND, resolveNotificationNavigationLocation } from './navigation';

describe('notification navigation resolver', () => {
  it('maps audit incidents to the existing audit incident route', () => {
    expect(
      resolveNotificationNavigationLocation({
        kind: NOTIFICATION_NAVIGATION_KIND.AUDIT_INCIDENT,
        payload: { incident_id: '101' },
      }),
    ).toEqual({
      path: '/audit/incidents/101',
    });
  });

  it('maps audit logs to existing audit log query contracts', () => {
    expect(
      resolveNotificationNavigationLocation({
        kind: NOTIFICATION_NAVIGATION_KIND.AUDIT_LOG,
        payload: { request_id: 'req-1' },
      }),
    ).toEqual({
      path: '/audit/logs',
      query: {
        request_id: 'req-1',
      },
    });
  });

  it('keeps scheduler run navigation on the scheduled task module route', () => {
    expect(
      resolveNotificationNavigationLocation({
        kind: NOTIFICATION_NAVIGATION_KIND.SCHEDULER_RUN,
        payload: { task_id: 'audit.retention', run_id: '7' },
      }),
    ).toEqual({
      path: '/server/scheduled-tasks',
      query: {
        task_key: 'audit.retention',
        run_id: '7',
      },
    });
  });

  it('keeps phase-two navigation kinds as placeholders', () => {
    expect(
      resolveNotificationNavigationLocation({
        kind: NOTIFICATION_NAVIGATION_KIND.SYSTEM_CONFIG_ITEM,
        payload: { config_key: 'auth.password.min_length' },
      }),
    ).toBeNull();

    expect(
      resolveNotificationNavigationLocation({
        kind: NOTIFICATION_NAVIGATION_KIND.MODULE_RUNTIME_ITEM,
        payload: { module_id: 'audit' },
      }),
    ).toBeNull();
  });
});
