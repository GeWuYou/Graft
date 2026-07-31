import { defineStore } from 'pinia';

import type { RealtimeTopicEventStreamController } from '@/shared/realtime/sse-client';

import { getUpdateOperationDiagnostic, subscribeToUpdateOperation } from '../api/update';
import type { UpdateFailureDiagnostic, UpdateOperation } from '../types/update';

type ProgressPhase = 'idle' | 'running' | 'reconnecting' | 'success' | 'failed';

const SESSION_STORAGE_KEY = 'graft.platform-update.operation-id';
const terminalSuccess = new Set<UpdateOperation['status']>(['SUCCESS', 'RECOVERED']);
const terminalFailure = new Set<UpdateOperation['status']>(['FAILED', 'NEEDS_ATTENTION']);

function restoreOperation() {
  try {
    const operationID = window.sessionStorage.getItem(SESSION_STORAGE_KEY)?.trim();
    return operationID ? ({ operation_id: operationID, status: 'PLANNING' } as UpdateOperation) : null;
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

/** 壳层唯一的升级会话状态：以 SSE 快照驱动，并在服务重建期间恢复连接。 */
export const useUpdateProgressStore = defineStore('update-progress', {
  state: () => {
    const operation = restoreOperation();
    return {
      operation,
      diagnostic: null as UpdateFailureDiagnostic | null,
      phase: (operation ? 'running' : 'idle') as ProgressPhase,
      lastActiveStatus: (operation?.status ?? null) as UpdateOperation['status'] | null,
      session: 0,
      stream: null as RealtimeTopicEventStreamController | null,
    };
  },
  getters: {
    visible: (state) => state.phase !== 'idle',
  },
  actions: {
    begin(operation: UpdateOperation) {
      this.session += 1;
      this.stopStream();
      persistOperation(operation.operation_id);
      this.operation = operation;
      this.lastActiveStatus = operation.status;
      this.diagnostic = null;
      this.phase = 'running';
      this.connect(this.session);
    },
    resume() {
      if (this.operation && !this.stream && this.phase !== 'idle' && !this.isTerminal()) {
        this.connect(this.session);
      }
    },
    connect(session: number) {
      const operationID = this.operation?.operation_id;
      if (!operationID || session !== this.session) return;
      this.stream = subscribeToUpdateOperation(operationID, {
        onOperation: async (operation) => this.applyOperation(session, operation),
        onStateChange: (state) => {
          if (session !== this.session || this.isTerminal()) return;
          this.phase = state === 'open' ? 'running' : state === 'idle' ? 'idle' : 'reconnecting';
        },
      });
    },
    async applyOperation(session: number, operation: UpdateOperation) {
      if (session !== this.session) return;
      this.operation = operation;
      if (terminalSuccess.has(operation.status)) {
        this.phase = 'success';
        this.stopStream();
        persistOperation(null);
        window.setTimeout(() => {
          if (session === this.session) window.location.reload();
        }, 1200);
        return;
      }
      if (terminalFailure.has(operation.status)) {
        this.phase = 'failed';
        this.stopStream();
        persistOperation(null);
        try {
          const diagnostic = await getUpdateOperationDiagnostic(operation.operation_id);
          if (session === this.session) this.diagnostic = diagnostic;
        } catch {
          if (session === this.session) this.diagnostic = null;
        }
        return;
      }
      this.lastActiveStatus = operation.status;
      this.phase = 'running';
    },
    isTerminal() {
      return this.phase === 'success' || this.phase === 'failed';
    },
    stopStream() {
      this.stream?.close();
      this.stream = null;
    },
    reset() {
      this.session += 1;
      this.stopStream();
      persistOperation(null);
      this.operation = null;
      this.lastActiveStatus = null;
      this.diagnostic = null;
      this.phase = 'idle';
    },
  },
});
