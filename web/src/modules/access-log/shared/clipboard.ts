import { MessagePlugin } from 'tdesign-vue-next/es/message';
import type { ComposerTranslation } from 'vue-i18n';

import { copyText } from '@/shared/observability';

/** 复制失败只反馈给当前交互，不让剪贴板能力影响日志详情的主流程。 */
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
