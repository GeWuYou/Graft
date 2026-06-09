// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import { listUserSessions, revokeAllUserSessions, revokeUserSession } from './user-sessions';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('user sessions api', () => {
  it('calls the canonical admin user-sessions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce([{ session_id: 'session-1', current: false }] as never);

    await listUserSessions(7, { limit: 2 });

    expect(requestGet).toHaveBeenCalledWith({
      url: '/api/users/7/sessions',
      params: {
        limit: 2,
      },
    });
  });

  it('absorbs the admin revoke-all envelope at the module api boundary', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(revokeAllUserSessions(7)).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: '/api/users/7/sessions/revoke-all',
    });
  });

  it('encodes the admin single-session revoke path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(revokeUserSession(7, 'session/with spaces')).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: '/api/users/7/sessions/session%2Fwith%20spaces/revoke',
    });
  });

  it('keeps the module contract templates canonical', () => {
    expect(USER_API_PATH.USER_SESSIONS_TEMPLATE).toBe('/api/users/{id}/sessions');
    expect(USER_API_PATH.USER_SESSIONS_REVOKE_ALL_TEMPLATE).toBe('/api/users/{id}/sessions/revoke-all');
    expect(USER_API_PATH.USER_SESSION_REVOKE_TEMPLATE).toBe('/api/users/{id}/sessions/{sessionID}/revoke');
  });
});
