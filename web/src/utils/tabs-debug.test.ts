import { beforeEach, describe, expect, it, vi } from 'vitest';

const { emitDebugLog, isDebugFlagEnabled } = vi.hoisted(() => ({
  emitDebugLog: vi.fn(),
  isDebugFlagEnabled: vi.fn<(flagId: string) => boolean>(),
}));

vi.mock('@/shared/debug/runtime', () => ({
  emitDebugLog,
  isDebugFlagEnabled,
}));

import { logTabsDebug } from './tabs-debug';

describe('tabs debug helper', () => {
  beforeEach(() => {
    emitDebugLog.mockReset();
    isDebugFlagEnabled.mockReset();
  });

  it('does not evaluate lazy messages when the debug flag is disabled', () => {
    isDebugFlagEnabled.mockReturnValue(false);
    const lazyMessage = vi.fn(() => 'should-not-run');

    logTabsDebug('tabs.store', lazyMessage);

    expect(lazyMessage).not.toHaveBeenCalled();
    expect(emitDebugLog).not.toHaveBeenCalled();
  });

  it('evaluates lazy messages only after the debug flag passes', () => {
    isDebugFlagEnabled.mockReturnValue(true);
    const lazyMessage = vi.fn(() => 'ready');

    logTabsDebug('tabs.layout', lazyMessage);

    expect(lazyMessage).toHaveBeenCalledTimes(1);
    expect(emitDebugLog).toHaveBeenCalledWith('tabs.layout', 'trace', {
      message: 'ready',
    });
  });
});
