import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

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
      url: OPENAPI_RUNTIME_PATH.getDashboardSummary,
    });
  });

  it('encodes widget ids for focused widget refresh', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 'audit.recent-events' } as never);

    await getDashboardWidget('audit.recent-events/error');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getDashboardWidget', { widget_id: 'audit.recent-events/error' }),
    });
  });
});
