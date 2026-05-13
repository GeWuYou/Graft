import { beforeEach, describe, expect, it } from 'vitest';

import { createTestingPinia } from '@/test/helpers';

import { useNavigationStore } from './navigation';

describe('navigation store', () => {
  beforeEach(() => {
    createTestingPinia();
  });

  it('returns the first accessible path for the current permissions', () => {
    const store = useNavigationStore();

    expect(store.firstAccessiblePath([])).toBe('');
    expect(store.firstAccessiblePath(['dashboard.view'])).toBe('/dashboard');
  });

  it('updates the active path explicitly', () => {
    const store = useNavigationStore();

    store.setActivePath('/dashboard');

    expect(store.activePath).toBe('/dashboard');
  });
});
