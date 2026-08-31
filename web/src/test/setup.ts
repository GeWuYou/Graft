import { config } from '@vue/test-utils';
import { afterEach, vi } from 'vitest';

const priorWarnHandler = config.global.config?.warnHandler;

config.global.config = {
  ...config.global.config,
  warnHandler(message, instance, trace) {
    if (message.includes('Failed to resolve component: t-')) {
      return;
    }

    if (priorWarnHandler) {
      priorWarnHandler(message, instance, trace);
    }
  },
};

// 每个测试后恢复 Vitest 全局状态，避免跨测试文件泄漏全局替身、间谍和假定时器。
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
