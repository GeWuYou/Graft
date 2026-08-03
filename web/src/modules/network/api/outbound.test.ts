import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  diagnoseOutboundNetwork,
  getOutboundNetworkDiagnosticHistory,
  getOutboundNetworkPolicy,
  resetOutboundNetworkPolicy,
  updateOutboundNetworkPolicy,
} from './outbound';

vi.mock('@/utils/request', () => ({
  request: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}));

describe('outbound network API', () => {
  it('uses the module-owned outbound policy and diagnostic routes', () => {
    const policy = { enabled: true, http_proxy: 'http://proxy:8080', https_proxy: '', no_proxy: ['localhost'] };

    getOutboundNetworkPolicy();
    updateOutboundNetworkPolicy(policy);
    resetOutboundNetworkPolicy();
    diagnoseOutboundNetwork('platform-update-release');
    getOutboundNetworkDiagnosticHistory('platform-update-release', 20);

    expect(request.get).toHaveBeenCalledWith({ url: '/api/platform/network/outbound' });
    expect(request.put).toHaveBeenCalledWith({ url: '/api/platform/network/outbound', data: policy });
    expect(request.post).toHaveBeenNthCalledWith(1, { url: '/api/platform/network/outbound/reset' });
    expect(request.post).toHaveBeenNthCalledWith(2, {
      url: '/api/platform/network/outbound/diagnostics/platform-update-release',
    });
    expect(request.get).toHaveBeenNthCalledWith(2, {
      url: '/api/platform/network/outbound/diagnostics/platform-update-release/history',
      params: { limit: 20 },
    });
  });
});
