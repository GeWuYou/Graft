// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

describe('route loading state', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
  });

  afterEach(async () => {
    const { hideRouteLoading } = await import('./route-loading');

    hideRouteLoading();
    vi.unstubAllGlobals();
    vi.useRealTimers();
    vi.resetModules();
  });

  it('keeps loading visible for the minimum display time after route render', async () => {
    const { ROUTE_LOADING_MIN_MS, finishRouteLoadingAfterRender, routeLoading, startRouteLoading } =
      await import('./route-loading');

    startRouteLoading();
    expect(routeLoading.value).toBe(true);

    await finishRouteLoadingAfterRender();
    expect(routeLoading.value).toBe(true);

    vi.advanceTimersByTime(ROUTE_LOADING_MIN_MS);
    expect(routeLoading.value).toBe(false);
  });

  it('uses the maximum timeout as a route-loading fallback', async () => {
    const { ROUTE_LOADING_MAX_MS, routeLoading, startRouteLoading } = await import('./route-loading');

    startRouteLoading();
    expect(routeLoading.value).toBe(true);

    vi.advanceTimersByTime(ROUTE_LOADING_MAX_MS);
    expect(routeLoading.value).toBe(false);
  });
});
