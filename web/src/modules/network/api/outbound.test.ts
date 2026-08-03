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
  request: {
    get: vi.fn(),
    getWithResponse: vi.fn(() => Promise.resolve({ data: {}, headers: {} })),
    post: vi.fn(),
    postWithResponse: vi.fn(() => Promise.resolve({ data: {}, headers: {} })),
    put: vi.fn(),
    putWithResponse: vi.fn(() => Promise.resolve({ data: {}, headers: {} })),
  },
}));

describe('outbound network API', () => {
  it('uses the module-owned outbound policy and diagnostic routes', () => {
    const policy = { enabled: true, http_proxy: 'http://proxy:8080', https_proxy: '', no_proxy: ['localhost'] };

    getOutboundNetworkPolicy();
    updateOutboundNetworkPolicy(policy, '"4"');
    resetOutboundNetworkPolicy('"5"');
    diagnoseOutboundNetwork('platform-update-release');
    getOutboundNetworkDiagnosticHistory('platform-update-release', 20);

    expect(request.getWithResponse).toHaveBeenCalledWith({ url: '/api/platform/network/outbound' });
    expect(request.putWithResponse).toHaveBeenCalledWith({
      url: '/api/platform/network/outbound',
      data: policy,
      headers: { 'If-Match': '"4"' },
    });
    expect(request.postWithResponse).toHaveBeenCalledWith({
      url: '/api/platform/network/outbound/reset',
      headers: { 'If-Match': '"5"' },
    });
    expect(request.post).toHaveBeenCalledWith({
      url: '/api/platform/network/outbound/diagnostics/platform-update-release',
    });
    expect(request.get).toHaveBeenCalledWith({
      url: '/api/platform/network/outbound/diagnostics/platform-update-release/history',
      params: { limit: 20 },
    });
  });
});
