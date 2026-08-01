import { defineStore } from 'pinia';

import type { RealtimeTopicEventStreamController } from '@/shared/realtime/sse-client';

import { getUpdateOperation, subscribeToUpdateOperation } from '../api/update';
import type { UpdateOperation, UpdateOperationLaunchAcknowledgement } from '../types/update';

type ProgressPhase = 'idle' | 'running' | 'reconnecting' | 'success' | 'failed';

const SESSION_STORAGE_KEY = 'graft.platform-update.operation-id';
const terminalSuccess = new Set<UpdateOperation['phase']>(['SUCCESS']);
const terminalFailure = new Set<UpdateOperation['phase']>(['FAILED', 'ROLLBACK']);
const POLL_INTERVAL_MS = 3000;

function restoreOperation() {
  try {
    const operationID = window.sessionStorage.getItem(SESSION_STORAGE_KEY)?.trim();
    return operationID || null;
  } catch {
    return null;
  }
}

function persistOperation(operationID: string | null) {
  try {
    if (operationID) window.sessionStorage.setItem(SESSION_STORAGE_KEY, operationID);
    else window.sessionStorage.removeItem(SESSION_STORAGE_KEY);
  } catch {
    // 禁用 storage 时仍保持当前标签页内的升级会话。
  }
}

/** 壳层唯一的升级会话状态：runner 快照优先于 SSE，服务重建时轮询同一状态事实。 */
export const useUpdateProgressStore = defineStore('update-progress', {
  state: () => {
    const operationID = restoreOperation();
    return {
      operation: null as UpdateOperation | null,
      operationID,
      phase: (operationID ? 'reconnecting' : 'idle') as ProgressPhase,
      lastActivePhase: null as UpdateOperation['phase'] | null,
      session: 0,
      stream: null as RealtimeTopicEventStreamController | null,
      pollTimer: null as number | null,
    };
  },
  getters: {
    visible: (state) => state.phase !== 'idle',
  },
  actions: {
    async begin(acknowledgement: UpdateOperationLaunchAcknowledgement) {
      this.session += 1;
      this.stopStream();
      this.stopPolling();
      persistOperation(acknowledgement.operation_id);
      this.operationID = acknowledgement.operation_id;
      this.operation = null;
      this.lastActivePhase = null;
      this.phase = 'reconnecting';
      await this.refreshSnapshot(this.session);
    },
    resume() {
      if (this.operationID && !this.stream && this.phase !== 'idle' && !this.isTerminal()) {
        void this.refreshSnapshot(this.session);
      }
    },
    async refreshSnapshot(session: number) {
      const operationID = this.operationID;
      if (!operationID || session !== this.session) return;
      try {
        const operation = await getUpdateOperation(operationID);
        await this.applyOperation(session, operation);
        if (session === this.session && !this.isTerminal() && !this.stream) this.connect(session);
      } catch {
        if (session !== this.session || this.isTerminal()) return;
        this.phase = 'reconnecting';
        this.startPolling(session);
      }
    },
    connect(session: number) {
      const operationID = this.operationID;
      if (!operationID || session !== this.session) return;
      this.stream = subscribeToUpdateOperation(operationID, {
        onOperation: async (operation) => this.applyOperation(session, operation),
        onStateChange: (state) => {
          if (session !== this.session || this.isTerminal()) return;
          if (state === 'open') {
            this.phase = 'running';
            this.stopPolling();
          } else if (state !== 'idle') {
            this.phase = 'reconnecting';
            this.startPolling(session);
          }
        },
      });
    },
    async applyOperation(session: number, operation: UpdateOperation) {
      if (session !== this.session) return;
      if (this.operationID && operation.operation_id !== this.operationID) return;
      this.operation = operation;
      this.operationID = operation.operation_id;
      if (terminalSuccess.has(operation.phase)) {
        this.phase = 'success';
        this.stopStream();
        this.stopPolling();
        persistOperation(null);
        this.operationID = null;
        window.setTimeout(() => {
          if (session === this.session) window.location.reload();
        }, 1200);
        return;
      }
      if (terminalFailure.has(operation.phase)) {
        this.phase = 'failed';
        this.stopStream();
        this.stopPolling();
        persistOperation(null);
        this.operationID = null;
        return;
      }
      this.lastActivePhase = operation.phase;
      this.phase = 'running';
    },
    isTerminal() {
      return this.phase === 'success' || this.phase === 'failed';
    },
    stopStream() {
      this.stream?.close();
      this.stream = null;
    },
    startPolling(session: number) {
      if (this.pollTimer !== null || session !== this.session) return;
      this.pollTimer = window.setInterval(() => void this.refreshSnapshot(session), POLL_INTERVAL_MS);
    },
    stopPolling() {
      if (this.pollTimer !== null) window.clearInterval(this.pollTimer);
      this.pollTimer = null;
    },
    reset() {
      this.session += 1;
      this.stopStream();
      this.stopPolling();
      persistOperation(null);
      this.operation = null;
      this.operationID = null;
      this.lastActivePhase = null;
      this.phase = 'idle';
    },
  },
});
