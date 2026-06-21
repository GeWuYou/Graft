import { afterEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { MONITOR_API_PATH } from '../contract/paths';
import { MONITOR_TREND_RANGE } from '../contract/trend';
import { getServerStatus } from './server-status';

const loggerMocks = vi.hoisted(() => ({
  debug: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warn: vi.fn(),
  child: vi.fn(),
  withContext: vi.fn(),
}));

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
  },
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => {
    loggerMocks.child.mockReturnValue(loggerMocks);
    loggerMocks.withContext.mockReturnValue(loggerMocks);
    return loggerMocks;
  },
}));

describe('monitor server-status api', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('calls the canonical monitor path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ status: 'healthy' } as never);

    const response = await getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES);

    expect(requestGet).toHaveBeenCalledWith({
      url: MONITOR_API_PATH.SERVER_STATUS,
      params: {
        trend_range: MONITOR_TREND_RANGE.TEN_MINUTES,
      },
    });
    expect(response).toEqual({ status: 'healthy' });
    expect(loggerMocks.warn).not.toHaveBeenCalled();
  });

  it('leaves responses unchanged when trend points are not an array', async () => {
    const requestGet = vi.mocked(request.get);
    const rawResponse = {
      status: 'healthy',
      trend: {
        points: null,
      },
    };
    requestGet.mockResolvedValueOnce(rawResponse as never);

    const response = await getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES);

    expect(response).toBe(rawResponse);
    expect(loggerMocks.warn).not.toHaveBeenCalled();
  });

  it('clamps finite out-of-range trend cpu_percent and logs warning context', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      status: 'healthy',
      trend: {
        points: [
          {
            observed_at: '2026-05-20T09:00:00Z',
            cpu_percent: 104.6,
          },
        ],
      },
    } as never);

    const response = await getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES);

    expect(response.trend.points[0]?.cpu_percent).toBe(100);
    expect(loggerMocks.warn).toHaveBeenCalledWith(
      'monitor server status trend cpu_percent out of range',
      expect.objectContaining({
        rawValue: 104.6,
        normalizedValue: 100,
        pointIndex: 0,
        observedAt: '2026-05-20T09:00:00Z',
        trendRange: MONITOR_TREND_RANGE.TEN_MINUTES,
      }),
    );
  });
});
