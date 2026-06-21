import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { AUTH_API_PATH } from '../contract/paths';
import {
  changePassword,
  completeRequiredPasswordChange,
  getBootstrap,
  listSessions,
  login,
  logout,
  refresh,
  revokeAllSessions,
  revokeOtherSessions,
  revokeSession,
} from './auth';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('auth api', () => {
  it('calls the canonical login path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'graft', password: 'admin' };
    requestPost.mockResolvedValueOnce({ access_token: 'token' } as never);

    await login(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.LOGIN,
      data: payload,
    });
  });

  it('calls the canonical bootstrap path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ user: { id: 1, username: 'graft', display_name: 'Graft' } } as never);

    await getBootstrap();

    expect(requestGet).toHaveBeenCalledWith({
      url: AUTH_API_PATH.BOOTSTRAP,
    });
  });

  it('calls the canonical refresh path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ access_token: 'rotated-token' } as never);

    await refresh();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.REFRESH,
    });
  });

  it('absorbs the logout envelope at the module api boundary', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(logout()).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.LOGOUT,
    });
  });

  it('calls the canonical sessions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce([{ session_id: 'session-1', current: true }] as never);

    await listSessions({ limit: 10 });

    expect(requestGet).toHaveBeenCalledWith({
      url: AUTH_API_PATH.SESSIONS,
      params: {
        limit: 10,
      },
    });
  });

  it('absorbs the revoke-all envelope at the module api boundary', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(revokeAllSessions()).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.SESSIONS_REVOKE_ALL,
    });
  });

  it('absorbs the revoke-others envelope at the module api boundary', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(revokeOtherSessions()).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.SESSIONS_REVOKE_OTHERS,
    });
  });

  it('encodes the session revoke path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(revokeSession('session/with spaces')).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: '/api/auth/sessions/session%2Fwith%20spaces/revoke',
    });
  });

  it('calls the canonical change-password path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { current_password: 'current-password-123', new_password: 'next-password-456' };
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(changePassword(payload)).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.CHANGE_PASSWORD,
      data: payload,
    });
  });

  it('calls the canonical complete-required-password-change path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { new_password: 'next-password-456' };
    requestPost.mockResolvedValueOnce(undefined as never);

    await expect(completeRequiredPasswordChange(payload)).resolves.toBeUndefined();

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.COMPLETE_REQUIRED_PASSWORD_CHANGE,
      data: payload,
    });
  });
});
