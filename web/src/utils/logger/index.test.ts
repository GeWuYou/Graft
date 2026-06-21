import { afterEach, describe, expect, it, vi } from 'vitest';

const { transportLog } = vi.hoisted(() => ({
  transportLog: vi.fn(),
}));

vi.mock('@/utils/logger/transports/consola', () => ({
  createConsolaTransport: () => ({
    log: transportLog,
  }),
}));

vi.mock('@/utils/logger/transports/noop', () => ({
  noopTransport: {
    log: vi.fn(),
  },
}));

async function loadLoggerModule() {
  vi.resetModules();
  return import('@/utils/logger');
}

describe('createLogger', () => {
  afterEach(() => {
    transportLog.mockReset();
    vi.unstubAllEnvs();
  });

  it('uses debug as the default level in development', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', '');

    const { createLogger } = await loadLoggerModule();
    createLogger('request').debug('debug message');

    expect(transportLog).toHaveBeenCalledTimes(1);
    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'debug',
        moduleName: 'request',
        message: 'debug message',
      }),
    );
  });

  it('uses warn as the default level in production', async () => {
    vi.stubEnv('DEV', false);
    vi.stubEnv('PROD', true);
    vi.stubEnv('VITE_LOG_LEVEL', '');

    const { createLogger } = await loadLoggerModule();
    const logger = createLogger('request');
    logger.info('info message');
    logger.warn('warn message');

    expect(transportLog).toHaveBeenCalledTimes(1);
    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'warn',
        message: 'warn message',
      }),
    );
  });

  it('falls back to the environment default when VITE_LOG_LEVEL is invalid', async () => {
    vi.stubEnv('DEV', false);
    vi.stubEnv('PROD', true);
    vi.stubEnv('VITE_LOG_LEVEL', 'verbose');

    const { createLogger } = await loadLoggerModule();
    createLogger('request').info('info message');

    expect(transportLog).not.toHaveBeenCalled();
  });

  it('does not dispatch logs when the resolved level is silent', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'silent');

    const { createLogger } = await loadLoggerModule();
    createLogger('request').error('silent message');

    expect(transportLog).not.toHaveBeenCalled();
  });

  it('builds child logger module names with a colon separator', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'debug');

    const { createLogger } = await loadLoggerModule();
    createLogger('request').child('auth').info('child message');

    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        moduleName: 'request:auth',
      }),
    );
  });

  it('merges context with per-call meta and lets per-call meta override same-name fields', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'debug');

    const { createLogger } = await loadLoggerModule();
    createLogger('etl')
      .withContext({
        taskId: 'task-1',
        shared: 'context',
      })
      .info('context message', {
        shared: 'meta',
        datasourceId: 'ds-1',
      });

    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        meta: {
          taskId: 'task-1',
          shared: 'meta',
          datasourceId: 'ds-1',
        },
      }),
    );
  });

  it('preserves the original Error object and stack when logging errors', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'debug');

    const error = new Error('request failed');
    const { createLogger } = await loadLoggerModule();
    createLogger('request').error(error, {
      requestId: 'req-1',
    });

    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'error',
        message: 'request failed',
        error,
        meta: {
          requestId: 'req-1',
        },
      }),
    );
    expect(error.stack).toBeTruthy();
  });

  it('does not fabricate an Error object when error receives a string message', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'debug');

    const { createLogger } = await loadLoggerModule();
    createLogger('request').error('request failed', {
      requestId: 'req-1',
    });

    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'error',
        message: 'request failed',
        meta: {
          requestId: 'req-1',
        },
      }),
    );
    expect(transportLog.mock.calls[0]?.[0]).not.toHaveProperty('error');
  });

  it('merges global context ahead of logger-local context and per-call meta', async () => {
    vi.stubEnv('DEV', true);
    vi.stubEnv('PROD', false);
    vi.stubEnv('VITE_LOG_LEVEL', 'debug');

    const { createLogger, patchGlobalLoggerContext } = await loadLoggerModule();
    patchGlobalLoggerContext({
      route: '/roles',
      requestId: 'req-7',
    });

    createLogger('request')
      .withContext({
        requestId: 'req-local',
      })
      .info('merged context', {
        traceId: 'trace-7',
      });

    expect(transportLog).toHaveBeenCalledWith(
      expect.objectContaining({
        meta: {
          route: '/roles',
          requestId: 'req-local',
          traceId: 'trace-7',
        },
      }),
    );
  });
});
