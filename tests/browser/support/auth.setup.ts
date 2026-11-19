import { request, type FullConfig } from '@playwright/test';

import { browserEnv } from './env';

// Global setup authenticates once through the real auth API and stores the
// session for every test. The harness runs `graft dev reset-admin` before the
// attempt, so the default admin always starts in the must-change-password state.
export default async function globalSetup(_config: FullConfig): Promise<void> {
  const api = await request.newContext({ baseURL: browserEnv.baseURL });
  const login = await api.post('/api/auth/login', {
    data: { username: browserEnv.adminUsername, password: browserEnv.adminPassword },
  });
  if (!login.ok()) {
    throw new Error(`browser fixture login failed with ${login.status()}`);
  }
  const change = await api.post('/api/auth/complete-required-password-change', {
    data: { new_password: browserEnv.acceptancePassword },
  });
  if (!change.ok()) {
    throw new Error(`required password change failed with ${change.status()}`);
  }
  // Re-login so the stored session reflects the acceptance password.
  const relogin = await api.post('/api/auth/login', {
    data: { username: browserEnv.adminUsername, password: browserEnv.acceptancePassword },
  });
  if (!relogin.ok()) {
    throw new Error(`browser fixture re-login failed with ${relogin.status()}`);
  }
  await api.storageState({ path: './test-results/auth-state.json' });
  await api.dispose();
}
