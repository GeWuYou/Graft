import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import {
  getProjectListSummaryTopicName,
  parseProjectListSummaryRealtimePayload,
  type ProjectListSummaryRealtimeItem,
} from '../contract/realtime';

type ProjectListRealtimeListener = (items: ProjectListSummaryRealtimeItem[]) => void;

let controller: RealtimeTopicSocketController | null = null;
let active = false;
const listeners = new Set<ProjectListRealtimeListener>();

function ensureSubscription() {
  if (!active || controller) {
    return;
  }
  controller = openRealtimeTopicSocket({
    topic: getProjectListSummaryTopicName(),
    parseMessage: parseProjectListSummaryRealtimePayload,
    onMessage: (message) => {
      listeners.forEach((listener) => listener(message.items));
    },
  });
}

function releaseSubscription() {
  controller?.close();
  controller = null;
}

export function acquireProjectListRealtime(listener: ProjectListRealtimeListener) {
  listeners.add(listener);
  active = true;
  ensureSubscription();
}

export function releaseProjectListRealtime(listener: ProjectListRealtimeListener) {
  listeners.delete(listener);
  if (listeners.size > 0) {
    return;
  }
  active = false;
  releaseSubscription();
}
