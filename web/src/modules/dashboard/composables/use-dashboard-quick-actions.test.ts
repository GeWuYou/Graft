// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import {
  DASHBOARD_QUICK_ACTION_STORAGE_KEY,
  DASHBOARD_QUICK_ACTION_STRATEGY,
  type DashboardQuickActionConfig,
} from '../contract/quick-actions';
import { useDashboardQuickActions } from './use-dashboard-quick-actions';

const loggerMocks = vi.hoisted(() => ({
  warn: vi.fn(),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => loggerMocks,
}));

const defaultConfig: DashboardQuickActionConfig = {
  enabled: true,
  maxItems: 8,
  strategy: DASHBOARD_QUICK_ACTION_STRATEGY.HYBRID,
};

function quickLink(index: number, partial: Partial<DashboardQuickActionLink> = {}): DashboardQuickActionLink {
  return {
    id: `link-${index}`,
    module_key: 'dashboard',
    order: index,
    route_location: `/route-${index}`,
    title: `Link ${index}`,
    ...partial,
  };
}

function useQuickActions(config: Partial<DashboardQuickActionConfig> = {}) {
  return useDashboardQuickActions(
    () => [quickLink(1), quickLink(2), quickLink(3)],
    () => ({ ...defaultConfig, ...config }),
  );
}

describe('useDashboardQuickActions', () => {
  beforeEach(() => {
    localStorage.clear();
    loggerMocks.warn.mockReset();
  });

  it('logs and ignores corrupt usage JSON from localStorage', () => {
    localStorage.setItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE, '{');

    const { rankedLinks } = useQuickActions();

    expect(rankedLinks.value.map((link) => link.id)).toEqual(['link-1', 'link-2', 'link-3']);
    expect(loggerMocks.warn).toHaveBeenCalledWith(
      'dashboard quick-action usage storage parse failed',
      expect.objectContaining({
        storageKey: DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE,
        error: expect.any(SyntaxError),
      }),
    );
  });

  it('logs and ignores non-object usage payloads from localStorage', () => {
    localStorage.setItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE, '[]');

    const { rankedLinks } = useQuickActions();

    expect(rankedLinks.value.map((link) => link.id)).toEqual(['link-1', 'link-2', 'link-3']);
    expect(loggerMocks.warn).toHaveBeenCalledWith(
      'dashboard quick-action usage storage payload invalid',
      expect.objectContaining({
        storageKey: DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE,
      }),
    );
  });

  it('normalizes invalid last access timestamps before ranking recent links', () => {
    localStorage.setItem(
      DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE,
      JSON.stringify({
        '/route-1': { accessCount: 1, lastAccessAt: 'not-a-date' },
        '/route-2': { accessCount: 1, lastAccessAt: '2026-06-09T12:00:00.000Z' },
      }),
    );

    const { rankedLinks } = useQuickActions({ strategy: DASHBOARD_QUICK_ACTION_STRATEGY.RECENT });

    expect(rankedLinks.value.map((link) => [link.route_location, link.lastAccessAt])).toEqual([
      ['/route-2', '2026-06-09T12:00:00.000Z'],
      ['/route-1', ''],
      ['/route-3', ''],
    ]);
  });
});
