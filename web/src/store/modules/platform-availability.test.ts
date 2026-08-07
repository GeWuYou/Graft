import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { fetchHealthz, setPlatformQueryOnline, setRealtimePlatformAvailable } = vi.hoisted(() => ({
  fetchHealthz: vi.fn(),
  setPlatformQueryOnline: vi.fn(),
  setRealtimePlatformAvailable: vi.fn(),
}));

vi.mock('@/utils/request', () => ({ probePlatformHealth: fetchHealthz, registerPlatformAvailabilityBridge: vi.fn() }));

vi.mock('@/shared/query/client', () => ({ setPlatformQueryOnline }));
vi.mock('@/shared/realtime/platform-availability', () => ({ setRealtimePlatformAvailable }));

import { usePlatformAvailabilityStore } from './platform-availability';

describe('platform availability store', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setActivePinia(createPinia());
    vi.stubGlobal('fetch', fetchHealthz);
    fetchHealthz.mockReset();
    setPlatformQueryOnline.mockReset();
    setRealtimePlatformAvailable.mockReset();
  });

  afterEach(() => {
    usePlatformAvailabilityStore().stopHealthMonitoring();
    vi.useRealTimers();
  });

  it('continues one healthz probe loop while unavailable and restores traffic after recovery', async () => {
    const availability = usePlatformAvailabilityStore();
    fetchHealthz.mockRejectedValueOnce(new Error('server offline')).mockResolvedValueOnce({ ok: true });

    await expect(availability.checkHealth()).resolves.toBe(false);
    expect(availability.status).toBe('unavailable');
    expect(setPlatformQueryOnline).toHaveBeenLastCalledWith(false);
    expect(setRealtimePlatformAvailable).toHaveBeenLastCalledWith(false);

    await vi.advanceTimersByTimeAsync(3_000);

    expect(fetchHealthz).toHaveBeenCalledTimes(2);
    expect(availability.status).toBe('healthy');
    expect(setPlatformQueryOnline).toHaveBeenLastCalledWith(true);
    expect(setRealtimePlatformAvailable).toHaveBeenLastCalledWith(true);
  });

  it('probes a healthy platform periodically instead of waiting for a business request to fail', async () => {
    const availability = usePlatformAvailabilityStore();
    fetchHealthz.mockResolvedValueOnce({ ok: true }).mockRejectedValueOnce(new Error('server offline'));

    await expect(availability.checkHealth()).resolves.toBe(true);
    await vi.advanceTimersByTimeAsync(10_000);

    expect(fetchHealthz).toHaveBeenCalledTimes(2);
    expect(availability.status).toBe('unavailable');
    expect(setPlatformQueryOnline).toHaveBeenLastCalledWith(false);
    expect(setRealtimePlatformAvailable).toHaveBeenLastCalledWith(false);
  });
});
