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

  it('accepts backend-owned alert counts', () => {
    expect(
      asAlertListPayload({
        items: [
          {
            action_label: 'View authentication failures',
            action_label_key: 'dashboard.widget.auditRiskEvents.authFailures.action',
            count: 4,
            id: 'risk.auth',
            level: 'warning',
            title: 'Authentication failures',
            title_key: 'audit.overview.riskGroups.authFailures',
          },
        ],
      }),
    ).not.toBeNull();
    expect(
      asAlertListPayload({
        items: [
          {
            count: 0,
            id: 'risk.auth',
            level: 'warning',
            title: 'Authentication failures',
            title_key: 'audit.overview.riskGroups.authFailures',
          },
        ],
      }),
    ).toBeNull();
  });

  it('accepts key-first alerts without fallback titles and rejects malformed optional titles', () => {
    expect(
      asAlertListPayload({
        items: [
          {
            id: 'audit.high-risk',
            level: 'error',
            title_key: 'dashboard.widget.auditRiskEvents.highRisk.title',
          },
        ],
      }),
    ).not.toBeNull();
    expect(
      asAlertListPayload({
        items: [
          {
            id: 'error.5xx',
            level: 'error',
            title: 500,
            title_key: 'dashboard.widget.accessLogRequestAttention.error',
          },
        ],
      }),
    ).toBeNull();
  });

  it('rejects malformed minimal item shapes', () => {
    expect(
      asStatGroupPayload({ items: [{ key: 'enabled', label_key: 'dashboard.enabled', label: 'Enabled' }] }),
    ).toBeNull();
    expect(
      asStatGroupPayload({
        items: [{ key: 'enabled', label_key: 'dashboard.enabled', label: 'Enabled', value: '1', tone: 'critical' }],
      }),
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
      asTimelinePayload({
        items: [
          {
            id: 'event-1',
            title_key: 'dashboard.event',
            title: 'Event',
            occurred_at: '2026-08-17T03:20:00Z',
            status: 'critical',
          },
        ],
      }),
    ).toBeNull();
    expect(
      asHealthPayload({
        summary: { status: 'offline' },
        items: [{ key: 'db', label_key: 'dashboard.db', label: 'Database', status: 'healthy' }],
      }),
    ).toBeNull();
  });
});
