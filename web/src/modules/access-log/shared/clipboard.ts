import { MessagePlugin } from 'tdesign-vue-next/es/message';
import type { ComposerTranslation } from 'vue-i18n';

import { copyText } from '@/shared/observability';

export async function copyAccessLogValue(value: string, t: ComposerTranslation) {
  try {
    const copied = await copyText(value);
    if (!copied) {
      MessagePlugin.error(t('accessLog.actions.copyFail'));
      return;
    }
    MessagePlugin.success(t('accessLog.actions.copySuccess'));
  } catch {
    MessagePlugin.error(t('accessLog.actions.copyFail'));
  }
}
