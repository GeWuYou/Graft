import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  createConnectivityCustomTarget,
  deleteConnectivityCustomTarget,
  getConnectivityCustomTargets,
  getConnectivityExport,
  getConnectivityTrace,
} from './connectivity';

vi.mock('@/utils/request', () => ({
  request: { delete: vi.fn(), get: vi.fn(), post: vi.fn() },
}));

describe('connectivity API', () => {
  it('keeps custom target management, trace, and export on target-addressed OpenAPI paths', () => {
    getConnectivityCustomTargets();
    createConnectivityCustomTarget({
      target_id: 'custom-status',
      display_name: 'Status',
      endpoint: 'https://example.com/health',
    });
    deleteConnectivityCustomTarget('custom-status');
    getConnectivityTrace('platform-update', 42);
    getConnectivityExport('platform-update', 42);

    expect(request.get).toHaveBeenCalledWith({ url: '/api/platform/network/connectivity/custom-targets' });
    expect(request.post).toHaveBeenCalledWith({
      url: '/api/platform/network/connectivity/custom-targets',
      data: { target_id: 'custom-status', display_name: 'Status', endpoint: 'https://example.com/health' },
    });
    expect(request.delete).toHaveBeenCalledWith({
      url: '/api/platform/network/connectivity/custom-targets/custom-status',
    });
    expect(request.get).toHaveBeenCalledWith({
      url: '/api/platform/network/connectivity/platform-update/reports/42/trace',
    });
    expect(request.get).toHaveBeenCalledWith({
      url: '/api/platform/network/connectivity/platform-update/reports/42/export',
    });
  });
});
