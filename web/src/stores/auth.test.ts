import { beforeEach, describe, expect, it } from 'vitest';

import { createTestingPinia } from '@/test/helpers';

import { useAuthStore } from './auth';

describe('auth store', () => {
  beforeEach(() => {
    window.localStorage.clear();
    createTestingPinia();
  });

  it('drops malformed persisted session payloads', () => {
    window.localStorage.setItem('graft:session', JSON.stringify({ token: 1 }));

    const store = useAuthStore();

    expect(store.isAuthenticated).toBe(false);
    expect(window.localStorage.getItem('graft:session')).toBeNull();
  });

  it('persists login state and clears it on logout', () => {
    const store = useAuthStore();

    store.login('admin');

    expect(store.isAuthenticated).toBe(true);
    expect(store.userName).toBe('admin');
    expect(window.localStorage.getItem('graft:session')).toContain(
      'mock-session-token',
    );

    store.logout();

    expect(store.isAuthenticated).toBe(false);
    expect(store.permissions).toEqual([]);
    expect(window.localStorage.getItem('graft:session')).toBeNull();
  });
});
