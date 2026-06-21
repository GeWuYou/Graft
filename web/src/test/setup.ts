// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
