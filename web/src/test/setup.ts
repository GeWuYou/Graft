import { config } from '@vue/test-utils';

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
