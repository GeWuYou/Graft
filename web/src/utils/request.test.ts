import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { API_CODE } from '@/api/model/authModel';

type MockConfig = Record<string, any>;
type MockResponse = {
  status: number;
  data: unknown;
  config?: MockConfig;
};
type MockError = {
  message?: string;
  response?: {
    status: number;
    data: unknown;
  };
  config?: MockConfig;
};

const { requestHandler } = vi.hoisted(() => ({
  requestHandler: vi.fn<(config: MockConfig) => Promise<MockResponse>>(),
}));

const { mockUserStore, locationReplace } = vi.hoisted(() => ({
  mockUserStore: {
    applyLoginResponse: vi.fn(),
    handleAuthFailure: vi.fn(),
  },
  locationReplace: vi.fn(),
}));

vi.mock('axios', () => {
  return {
    default: {
      create: () => {
        const requestInterceptors: Array<(config: MockConfig) => MockConfig | Promise<MockConfig>> = [];
        const responseInterceptors: Array<{
          fulfilled?: (response: unknown) => unknown | Promise<unknown>;
          rejected?: (error: MockError) => unknown | Promise<unknown>;
        }> = [];

        return {
          interceptors: {
            request: {
              use: (fulfilled: (config: MockConfig) => MockConfig | Promise<MockConfig>) => {
                requestInterceptors.push(fulfilled);
                return requestInterceptors.length - 1;
              },
            },
            response: {
              use: (
                fulfilled?: (response: unknown) => unknown | Promise<unknown>,
                rejected?: (error: MockError) => unknown | Promise<unknown>,
              ) => {
                responseInterceptors.push({ fulfilled, rejected });
                return responseInterceptors.length - 1;
              },
            },
          },
          async request(config: MockConfig) {
            let nextConfig = config;
            for (const interceptor of requestInterceptors) {
              nextConfig = await interceptor(nextConfig);
            }

            try {
              let response = await requestHandler(nextConfig);
              response = {
                ...response,
                config: response.config ?? nextConfig,
              };

              let current: unknown = response;
              for (const interceptor of responseInterceptors) {
                if (interceptor.fulfilled) {
                  current = await interceptor.fulfilled(current);
                }
              }
              return current;
            } catch (error) {
              let currentError: MockError = {
                ...(error as MockError),
                config: (error as MockError)?.config ?? nextConfig,
              };
              for (const interceptor of responseInterceptors) {
                if (!interceptor.rejected) {
                  continue;
                }

                try {
                  return await interceptor.rejected(currentError);
                } catch (nextError) {
                  currentError = nextError as MockError;
                }
              }
              throw currentError;
            }
          },
        };
      },
    },
  };
});

async function loadRequestModule() {
  vi.resetModules();
  return import('./request');
}

function createApiError(code: string, status = 401, message = code, traceId = 'trace-1') {
  return {
    message,
    response: {
      status,
      data: {
        success: false,
        code,
        message,
        traceId,
      },
    },
  };
}

describe('request auth handling', () => {
  beforeEach(() => {
    requestHandler.mockReset();
    mockUserStore.applyLoginResponse.mockReset();
    mockUserStore.handleAuthFailure.mockReset();
    locationReplace.mockReset();
    localStorage.clear();
    mockUserStore.applyLoginResponse.mockImplementation(async (payload: { access_token: string }) => {
      const { setAccessToken } = await import('@/utils/auth-state');
      setAccessToken(payload.access_token);
      localStorage.setItem('user', JSON.stringify({ token: payload.access_token }));
    });
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...window.location,
        pathname: '/',
        search: '',
        hash: '',
        replace: locationReplace,
      },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('only refreshes on AUTH_TOKEN_EXPIRED and replays the original request with the new token', async () => {
    const { registerAuthSessionBridge, request } = await loadRequestModule();
    const { setAccessToken } = await import('@/utils/auth-state');
    registerAuthSessionBridge(mockUserStore);

    const callLog: MockConfig[] = [];
    requestHandler.mockImplementation(async (config) => {
      callLog.push(config);

      if (config.url === '/api/users' && !config._authRefreshAttempted) {
        throw createApiError(API_CODE.AUTH_TOKEN_EXPIRED, 401, 'expired', 'trace-expired');
      }
      if (config.url === '/api/auth/refresh') {
        return {
          status: 200,
          data: {
            success: true,
            code: API_CODE.OK,
            message: 'OK',
            traceId: 'trace-refresh',
            data: {
              access_token: 'fresh-token',
            },
          },
        };
      }
      if (config.url === '/api/users' && config._authRefreshAttempted) {
        return {
          status: 200,
          data: {
            success: true,
            code: API_CODE.OK,
            message: 'OK',
            traceId: 'trace-users',
            data: {
              ok: true,
            },
          },
        };
      }

      throw new Error(`unexpected request ${String(config.url)}`);
    });

    setAccessToken('stale-token');
    localStorage.setItem('user', JSON.stringify({ token: 'stale-token' }));

    await expect(request.get<{ ok: boolean }>({ url: '/api/users' })).resolves.toEqual({ ok: true });

    expect(callLog).toHaveLength(3);
    expect(callLog[0]?.headers?.Authorization).toMatch(/^Bearer /);
    expect(callLog[1]?.url).toBe('/api/auth/refresh');
    expect(callLog[2]?.url).toBe('/api/users');
    expect(callLog[2]?._authRefreshAttempted).toBe(true);
    expect(mockUserStore.applyLoginResponse).toHaveBeenCalledWith(
      expect.objectContaining({
        access_token: 'fresh-token',
      }),
    );
  });

  it.each([API_CODE.AUTH_TOKEN_INVALID, API_CODE.AUTH_TOKEN_MISSING])(
    'clears the client session and redirects to login on %s',
    async (code) => {
      const { registerAuthSessionBridge, request } = await loadRequestModule();
      const { setAccessToken } = await import('@/utils/auth-state');
      registerAuthSessionBridge(mockUserStore);
      const callUrls: string[] = [];
      Object.assign(window.location, {
        pathname: '/users',
        search: '?tab=active',
        hash: '#detail',
      });

      requestHandler.mockImplementation(async (config) => {
        callUrls.push(String(config.url));
        throw createApiError(code, 401, code, 'trace-auth');
      });

      setAccessToken('stale-token');
      localStorage.setItem('user', JSON.stringify({ token: 'stale-token' }));

      await expect(request.get({ url: '/api/users' })).rejects.toMatchObject({
        code,
        status: 401,
      });

      expect(mockUserStore.handleAuthFailure).toHaveBeenCalledTimes(1);
      expect(callUrls).toEqual(['/api/users']);
      expect(callUrls).not.toContain('/api/auth/refresh');
      expect(locationReplace).toHaveBeenCalledWith('/login?redirect=%2Fusers%3Ftab%3Dactive%23detail');
    },
  );

  it('does not recursively refresh when the refresh request itself fails', async () => {
    const { registerAuthSessionBridge, request } = await loadRequestModule();
    const { setAccessToken } = await import('@/utils/auth-state');
    registerAuthSessionBridge(mockUserStore);
    Object.assign(window.location, {
      pathname: '/users',
      search: '',
      hash: '',
    });

    const callUrls: string[] = [];
    requestHandler.mockImplementation(async (config) => {
      callUrls.push(String(config.url));

      if (config.url === '/api/users') {
        throw createApiError(API_CODE.AUTH_TOKEN_EXPIRED, 401, 'expired', 'trace-expired');
      }
      if (config.url === '/api/auth/refresh') {
        throw createApiError(API_CODE.AUTH_TOKEN_EXPIRED, 401, 'expired', 'trace-refresh');
      }

      throw new Error(`unexpected request ${String(config.url)}`);
    });

    setAccessToken('stale-token');
    localStorage.setItem('user', JSON.stringify({ token: 'stale-token' }));
    window.history.pushState({}, '', '/users');

    await expect(request.get({ url: '/api/users' })).rejects.toMatchObject({
      code: API_CODE.AUTH_TOKEN_EXPIRED,
      status: 401,
    });

    expect(callUrls).toEqual(['/api/users', '/api/auth/refresh']);
    expect(mockUserStore.handleAuthFailure).toHaveBeenCalledTimes(1);
    expect(locationReplace).toHaveBeenCalledWith('/login?redirect=%2Fusers');
  });
});
