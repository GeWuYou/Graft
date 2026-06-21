import { describe, expect, it, vi } from 'vitest';

import { formatDashboardDateTime, openDashboardRoute } from './widget-actions';

describe('dashboard widget actions', () => {
  it('keeps dashboard route navigation delegated to router.push', () => {
    const router = {
      push: vi.fn(),
    };

    openDashboardRoute(router as never, '/access-control/roles');

    expect(router.push).toHaveBeenCalledWith('/access-control/roles');
  });

  it('falls back to the original value when date formatting receives an invalid date', () => {
    expect(formatDashboardDateTime('not-a-date')).toBe('not-a-date');
    expect(() => formatDashboardDateTime('not-a-date')).not.toThrow();
  });

  it('formats dashboard dates with the provided locale', () => {
    expect(formatDashboardDateTime('2026-06-10T02:38:00Z', 'en-US')).toBe(
      new Intl.DateTimeFormat('en-US', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date('2026-06-10T02:38:00Z')),
    );
  });
});
