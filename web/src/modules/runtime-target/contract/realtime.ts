import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

import type { RuntimeTarget } from '../api/runtime-target';

export const RUNTIME_TARGET_REALTIME_TOPIC = 'runtime-target.summary.list' as const;

type RuntimeTargetSummaryRealtimeMessage = {
  topic: typeof RUNTIME_TARGET_REALTIME_TOPIC;
  items: RuntimeTarget[];
};

/**
 * 解析并校验运行目标摘要实时消息。
 *
 * @param raw - 待解析的原始输入
 * @returns 符合运行目标摘要消息格式的数据；输入无效时返回 `null`
 */
export function parseRuntimeTargetSummaryPayload(raw: unknown): RuntimeTargetSummaryRealtimeMessage | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data) || data.topic !== RUNTIME_TARGET_REALTIME_TOPIC || !Array.isArray(data.items))
    return null;
  return data as RuntimeTargetSummaryRealtimeMessage;
}
