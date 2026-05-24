import { describe, expect, it, vi } from 'vitest';

const moduleAuthStore = vi.hoisted(() => ({
  useAuthSessionStore: vi.fn(),
}));

vi.mock('@/modules/auth/store/session', () => moduleAuthStore);

describe('useUserStore compatibility bridge', () => {
  it('re-exports the module-owned auth session store', async () => {
    const { useUserStore } = await import('./user');

    expect(useUserStore).toBe(moduleAuthStore.useAuthSessionStore);
  });
});
