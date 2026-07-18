import { describe, expect, it, vi } from 'vitest';

const { consolaMocks, createConsolaMock } = vi.hoisted(() => {
  const consolaMocks = {
    debug: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    withTag: vi.fn(),
  };

  return {
    consolaMocks,
    createConsolaMock: vi.fn(() => consolaMocks),
  };
});

consolaMocks.withTag.mockReturnValue(consolaMocks);

vi.mock('consola', () => ({
  createConsola: createConsolaMock,
}));

import { createConsolaTransport } from './consola';

describe('consola transport', () => {
  it('delegates level filtering to the application logger', () => {
    expect(createConsolaMock).toHaveBeenCalledWith({ level: Number.POSITIVE_INFINITY });
  });

  it('renders colored consola output as one compact line with flattened fields', () => {
    createConsolaTransport().log({
      level: 'error',
      moduleName: 'project.detail',
      message: 'request failed\nwhile loading details',
      timestamp: new Date('2026-07-15T06:30:00.000Z'),
      meta: {
        requestId: 'req-7',
        response: {
          status: 500,
        },
      },
      error: new Error('network\nfailed'),
    });

    expect(consolaMocks.withTag).toHaveBeenCalledWith('project.detail');
    expect(consolaMocks.error).toHaveBeenCalledWith(
      '2026-07-15T06:30:00.000Z | request failed while loading details | requestId="req-7" response.status=500 error.name="Error" error.message="network failed"',
    );
  });

  it('labels an Error passed as meta without an empty field prefix', () => {
    createConsolaTransport().log({
      level: 'warn',
      moduleName: 'project.detail',
      message: 'request failed',
      timestamp: new Date('2026-07-15T06:30:00.000Z'),
      meta: new Error('network failed'),
    });

    expect(consolaMocks.warn).toHaveBeenCalledWith(
      '2026-07-15T06:30:00.000Z | request failed | error.name="Error" error.message="network failed"',
    );
  });

  it('falls back safely when JSON.stringify omits function or symbol values', () => {
    createConsolaTransport().log({
      level: 'info',
      moduleName: 'project.detail',
      message: 'callback skipped',
      timestamp: new Date('2026-07-15T06:30:00.000Z'),
      meta: {
        callback: () => undefined,
        marker: Symbol('marker'),
        missing: undefined,
      },
    });

    expect(consolaMocks.info).toHaveBeenCalledWith(
      '2026-07-15T06:30:00.000Z | callback skipped | callback="[unserializable]" marker="[unserializable]" missing=undefined',
    );
  });
});
