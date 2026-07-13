import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

import type { RuntimeTarget } from '../api/runtime-target';

export const RUNTIME_TARGET_REALTIME_TOPIC = 'runtime-target.summary.list' as const;

type RuntimeTargetSummaryRealtimeMessage = {
  topic: typeof RUNTIME_TARGET_REALTIME_TOPIC;
  items: RuntimeTarget[];
};

export function parseRuntimeTargetSummaryPayload(raw: unknown): RuntimeTargetSummaryRealtimeMessage | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data) || data.topic !== RUNTIME_TARGET_REALTIME_TOPIC || !Array.isArray(data.items))
    return null;
  return data as RuntimeTargetSummaryRealtimeMessage;
}
