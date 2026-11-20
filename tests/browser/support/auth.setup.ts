import { mkdirSync, writeFileSync } from 'node:fs';

import { request, type FullConfig } from '@playwright/test';

import { browserEnv } from './env';

type LoginPayload = {
  data?: {
    access_token?: string;
    expires_at?: string;
    must_change_password?: boolean;
  };
};

type StorageStateShape = {
  cookies: Array<Record<string, unknown>>;
  origins: Array<{ origin: string; localStorage: Array<{ name: string; value: string }> }>;
};

// Global setup authenticates through the real auth API and stores the session
// for every test. The harness runs `graft dev reset-admin` when it provisions a
// fresh environment, so the default admin starts in the must-change-password
// state with the initial credential. Environments kept with --keep (and every
// attempt after the first one) already use the acceptance password, so the
// setup falls back instead of assuming one fixed password.
export default async function globalSetup(_config: FullConfig): Promise<void> {
  const api = await request.newContext({ baseURL: browserEnv.baseURL });

  const acceptanceLogin = await api.post('/api/auth/login', {
    data: { username: browserEnv.adminUsername, password: browserEnv.acceptancePassword },
  });
  if (!acceptanceLogin.ok()) {
    // The initial admin credential is still active; complete the forced
    // password change through the authenticated route.
    const initialLogin = await api.post('/api/auth/login', {
      data: { username: browserEnv.adminUsername, password: browserEnv.adminPassword },
    });
    if (!initialLogin.ok()) {
      throw new Error(`browser fixture login failed with ${initialLogin.status()}`);
    }
    const initialPayload = (await initialLogin.json()) as LoginPayload;
    const accessToken = initialPayload.data?.access_token;
    if (!accessToken) {
      throw new Error('login response did not include an access token');
    }
    const change = await api.post('/api/auth/complete-required-password-change', {
      headers: { Authorization: `Bearer ${accessToken}` },
      data: { new_password: browserEnv.acceptancePassword },
    });
    if (!change.ok()) {
      throw new Error(`required password change failed with ${change.status()}`);
    }
  }

  // Re-login with the acceptance password so the stored session matches the
  // credential every later attempt reuses.
  const sessionLogin = await api.post('/api/auth/login', {
    data: { username: browserEnv.adminUsername, password: browserEnv.acceptancePassword },
  });
  if (!sessionLogin.ok()) {
    throw new Error(`browser fixture session login failed with ${sessionLogin.status()}`);
  }
  const sessionPayload = (await sessionLogin.json()) as LoginPayload;
  const state = (await api.storageState()) as StorageStateShape;
  // The web client sends Authorization headers from its persisted session
  // snapshot (STORAGE_KEY.USER_SESSION), so seed it alongside the cookies.
  state.origins.push({
    origin: browserEnv.baseURL,
    localStorage: [
      {
        name: 'user',
        value: JSON.stringify({
          token: sessionPayload.data?.access_token ?? '',
          expiresAt: sessionPayload.data?.expires_at ?? '',
        }),
      },
    ],
  });
  mkdirSync('./test-results', { recursive: true });
  writeFileSync('./test-results/auth-state.json', JSON.stringify(state));
  await api.dispose();
}
