import { RUNTIME_TARGET_REALTIME_TOPIC } from '@/contracts/generated/modules/runtime-target';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

import type { RuntimeTarget } from '../api/runtime-target';

type RuntimeTargetSummaryRealtimeMessage = {
  topic: typeof RUNTIME_TARGET_REALTIME_TOPIC.SUMMARY;
  items: RuntimeTarget[];
};

/** WebSocket 消息属于外部输入；只接受当前主题和完整字段，避免用旧摘要污染详情页状态。 */
export function parseRuntimeTargetSummaryPayload(raw: unknown): RuntimeTargetSummaryRealtimeMessage | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (
    !isRealtimePayloadObject(data) ||
    data.topic !== RUNTIME_TARGET_REALTIME_TOPIC.SUMMARY ||
    !Array.isArray(data.items)
  )
    return null;
  return data as RuntimeTargetSummaryRealtimeMessage;
}
