import { beforeEach, describe, expect, it, vi } from 'vitest';

const healPersistedState = vi.fn();
const afterEachMock = vi.fn();
const registerRouteGuards = vi.fn();
const registerPermissionDirective = vi.fn();
const isHandledAuthRequestError = vi.fn();
const isProjectMonacoBenignCancellationError = vi.fn();
const loggerError = vi.fn();
const patchGlobalLoggerContext = vi.fn();
const useMock = vi.fn();
const mountMock = vi.fn();

vi.mock('vue', () => ({
  createApp: () => ({
    config: {},
    use: useMock,
    mount: mountMock,
  }),
}));

vi.mock('@/App.vue', () => ({
  default: {},
}));

vi.mock('@/router', () => ({
  default: {
    currentRoute: {
      value: {
        path: '/login',
      },
    },
    afterEach: afterEachMock,
  },
}));

vi.mock('@/locales', () => ({
  i18n: {},
}));

vi.mock('@/store', () => ({
  store: {},
  useTabsRouterStore: () => ({
    healPersistedState,
  }),
}));

vi.mock('@/utils/auth-request-error', () => ({
  isHandledAuthRequestError,
}));

vi.mock('@/modules/project/shared/project-monaco-debug', () => ({
  isProjectMonacoBenignCancellationError,
}));

vi.mock('./route-guards', () => ({
  registerRouteGuards,
}));

vi.mock('./permission-directive', () => ({
  registerPermissionDirective,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    withContext: () => ({
      error: loggerError,
    }),
  }),
  patchGlobalLoggerContext,
}));

describe('bootstrapApp', () => {
  beforeEach(() => {
    vi.resetModules();
    afterEachMock.mockReset();
    healPersistedState.mockReset();
    registerRouteGuards.mockReset();
    registerPermissionDirective.mockReset();
    isHandledAuthRequestError.mockReset();
    isHandledAuthRequestError.mockReturnValue(false);
    isProjectMonacoBenignCancellationError.mockReset();
    isProjectMonacoBenignCancellationError.mockReturnValue(false);
    loggerError.mockReset();
    patchGlobalLoggerContext.mockReset();
    useMock.mockReset();
    mountMock.mockReset();
  });

  it('heals persisted tab refresh residue before mounting the app', async () => {
    const { bootstrapApp } = await import('./index');

    bootstrapApp();

    expect(registerRouteGuards).toHaveBeenCalledTimes(1);
    expect(afterEachMock).toHaveBeenCalledTimes(1);
    expect(patchGlobalLoggerContext).toHaveBeenCalledWith({
      route: '/login',
    });
    expect(healPersistedState).toHaveBeenCalledTimes(1);
    expect(useMock).toHaveBeenCalledTimes(3);
    expect(useMock.mock.invocationCallOrder[0]).toBeLessThan(healPersistedState.mock.invocationCallOrder[0]);
    expect(registerPermissionDirective).toHaveBeenCalledTimes(1);
    expect(mountMock).toHaveBeenCalledWith('#app');
    expect(healPersistedState.mock.invocationCallOrder[0]).toBeLessThan(mountMock.mock.invocationCallOrder[0]);
  });

  it('suppresses Monaco cancellation rejections from the global runtime error sink', async () => {
    const { bootstrapApp } = await import('./index');
    isProjectMonacoBenignCancellationError.mockReturnValue(true);

    bootstrapApp();

    const event = new Event('unhandledrejection', {
      cancelable: true,
    });
    const reason = Object.assign(new Error('Canceled'), {
      name: 'Canceled',
      stack: 'Error: Canceled\n    at ProjectMonacoSurface.vue:124:1',
    });
    Object.defineProperty(event, 'reason', {
      configurable: true,
      value: reason,
    });

    window.dispatchEvent(event);

    expect(isProjectMonacoBenignCancellationError).toHaveBeenCalledWith(reason);
    expect(event.defaultPrevented).toBe(true);
    expect(loggerError).not.toHaveBeenCalled();
  });
});
