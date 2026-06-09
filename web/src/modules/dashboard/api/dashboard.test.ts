// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { buildDashboardWidgetApiPath, DASHBOARD_API_PATH } from '../contract/paths';
import { getDashboardSummary, getDashboardWidget } from './dashboard';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
  },
}));

describe('dashboard api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reads the dashboard summary from the canonical OpenAPI path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ system_summary: {}, widgets: [] } as never);

    await getDashboardSummary();

    expect(requestGet).toHaveBeenCalledWith({
      url: DASHBOARD_API_PATH.SUMMARY,
    });
  });

  it('encodes widget ids for focused widget refresh', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 'audit.recent-events' } as never);

    await getDashboardWidget('audit.recent-events/error');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildDashboardWidgetApiPath('audit.recent-events/error'),
    });
    expect(buildDashboardWidgetApiPath('audit.recent-events/error')).toBe(
      '/api/dashboard/widgets/audit.recent-events%2Ferror',
    );
  });
});
