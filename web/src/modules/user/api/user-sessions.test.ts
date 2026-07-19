import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

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

  it('builds session paths from generated operation ids', () => {
    expect(buildOpenApiRuntimePath('getUserSessions', { id: 7 })).toBe('/api/users/7/sessions');
    expect(buildOpenApiRuntimePath('postUserSessionsRevokeAll', { id: 7 })).toBe('/api/users/7/sessions/revoke-all');
    expect(buildOpenApiRuntimePath('postUserSessionRevoke', { id: 7, sessionID: 'session-1' })).toBe(
      '/api/users/7/sessions/session-1/revoke',
    );
  });
});
