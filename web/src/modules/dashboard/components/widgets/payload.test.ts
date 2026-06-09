// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import {
  asAlertListPayload,
  asHealthPayload,
  asLinkListPayload,
  asStatGroupPayload,
  asTimelinePayload,
} from './payload';

describe('dashboard widget payload guards', () => {
  it('accepts valid empty item payloads', () => {
    expect(asStatGroupPayload({ items: [] })).not.toBeNull();
    expect(asAlertListPayload({ items: [] })).not.toBeNull();
    expect(asLinkListPayload({ items: [] })).not.toBeNull();
    expect(asTimelinePayload({ items: [] })).not.toBeNull();
    expect(asHealthPayload({ summary: { status: 'healthy' }, items: [] })).not.toBeNull();
  });

  it('rejects malformed minimal item shapes', () => {
    expect(
      asStatGroupPayload({ items: [{ key: 'enabled', label_key: 'dashboard.enabled', label: 'Enabled' }] }),
    ).toBeNull();
    expect(
      asAlertListPayload({ items: [{ id: 'risk-1', level: 'critical', title_key: 'dashboard.risk', title: 'Risk' }] }),
    ).toBeNull();
    expect(
      asLinkListPayload({
        items: [{ key: 'roles', label_key: 'dashboard.roles', route_location: '/rbac/roles' }],
      }),
    ).toBeNull();
    expect(asTimelinePayload({ items: [{ id: 'event-1', title_key: 'dashboard.event', title: 'Event' }] })).toBeNull();
    expect(
      asHealthPayload({
        summary: { status: 'offline' },
        items: [{ key: 'db', label_key: 'dashboard.db', label: 'Database', status: 'healthy' }],
      }),
    ).toBeNull();
  });
});
