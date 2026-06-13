// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { MessagePlugin } from 'tdesign-vue-next/es/message';

import { copyText } from '@/shared/observability';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export async function copyAuditRequestId(
  requestId: string,
  t: Translate,
  options: {
    warnWhenMissing?: boolean;
  } = {},
) {
  if (!requestId || requestId === '-') {
    if (options.warnWhenMissing) {
      MessagePlugin.warning(t('audit.logList.drawer.actions.copyRequestIdFail'));
    }
    return false;
  }

  try {
    const copied = await copyText(requestId);
    if (!copied) {
      MessagePlugin.error(t('audit.logList.drawer.actions.copyRequestIdFail'));
      return false;
    }
    MessagePlugin.success(t('audit.logList.drawer.actions.copyRequestIdSuccess'));
    return true;
  } catch {
    MessagePlugin.error(t('audit.logList.drawer.actions.copyRequestIdFail'));
    return false;
  }
}
