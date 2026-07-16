import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import {
  type ApplicationListSummaryRealtimeItem,
  getApplicationListSummaryTopicName,
  parseApplicationListSummaryRealtimePayload,
} from '../contract/realtime';

type ApplicationListRealtimeListener = (items: ApplicationListSummaryRealtimeItem[]) => void;

let controller: RealtimeTopicSocketController | null = null;
let active = false;
const listeners = new Set<ApplicationListRealtimeListener>();

function ensureSubscription() {
  if (!active || controller) {
    return;
  }
  controller = openRealtimeTopicSocket({
    topic: getApplicationListSummaryTopicName(),
    parseMessage: parseApplicationListSummaryRealtimePayload,
    onMessage: (message) => {
      listeners.forEach((listener) => listener(message.items));
    },
  });
}

function releaseSubscription() {
  controller?.close();
  controller = null;
}

export function acquireApplicationListRealtime(listener: ApplicationListRealtimeListener) {
  listeners.add(listener);
  active = true;
  ensureSubscription();
}

export function releaseApplicationListRealtime(listener: ApplicationListRealtimeListener) {
  listeners.delete(listener);
  if (listeners.size > 0) {
    return;
  }
  active = false;
  releaseSubscription();
}
