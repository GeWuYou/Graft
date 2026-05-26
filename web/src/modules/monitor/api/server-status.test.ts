import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { MONITOR_API_PATH } from '../contract/paths';
import { MONITOR_TREND_RANGE } from '../contract/trend';
import { getServerStatus } from './server-status';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
  },
}));

describe('monitor server-status api', () => {
  it('calls the canonical monitor path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ status: 'healthy' } as never);

    await getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES);

    expect(requestGet).toHaveBeenCalledWith({
      url: MONITOR_API_PATH.SERVER_STATUS,
      params: {
        trend_range: MONITOR_TREND_RANGE.TEN_MINUTES,
      },
    });
  });
});
