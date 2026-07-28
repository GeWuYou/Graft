import { defineStore } from 'pinia';

import { getUpdateOperation, getUpdateOperationDiagnostic } from '../api/update';
import type { UpdateFailureDiagnostic, UpdateOperation } from '../types/update';

type ProgressPhase = 'idle' | 'running' | 'reconnecting' | 'success' | 'failed';

const terminalSuccess = new Set<UpdateOperation['status']>(['SUCCESS', 'RECOVERED']);
const terminalFailure = new Set<UpdateOperation['status']>(['FAILED', 'NEEDS_ATTENTION']);

/** 壳层唯一的升级会话状态，服务重建期间继续保留并重连操作查询。 */
export const useUpdateProgressStore = defineStore('update-progress', {
  state: () => ({
    operation: null as UpdateOperation | null,
    diagnostic: null as UpdateFailureDiagnostic | null,
    phase: 'idle' as ProgressPhase,
    pollTimer: null as number | null,
  }),
  getters: {
    visible: (state) => state.phase !== 'idle',
  },
  actions: {
    begin(operation: UpdateOperation) {
      this.stopPolling();
      this.operation = operation;
      this.diagnostic = null;
      this.phase = 'running';
      void this.poll();
    },
    async poll() {
      const operationID = this.operation?.operation_id;
      if (!operationID || this.phase === 'idle') return;
      try {
        const operation = await getUpdateOperation(operationID);
        this.operation = operation;
        if (terminalSuccess.has(operation.status)) {
          this.phase = 'success';
          this.stopPolling();
          window.setTimeout(() => window.location.reload(), 1200);
          return;
        }
        if (terminalFailure.has(operation.status)) {
          this.phase = 'failed';
          this.stopPolling();
          try {
            this.diagnostic = await getUpdateOperationDiagnostic(operationID);
          } catch {
            this.diagnostic = null;
          }
          return;
        }
        this.phase = 'running';
      } catch {
        // Compose 重建会短暂中断 API；保留遮罩并继续轮询。
        this.phase = 'reconnecting';
      }
      this.pollTimer = window.setTimeout(() => void this.poll(), 2000);
    },
    stopPolling() {
      if (this.pollTimer !== null) window.clearTimeout(this.pollTimer);
      this.pollTimer = null;
    },
    reset() {
      this.stopPolling();
      this.operation = null;
      this.diagnostic = null;
      this.phase = 'idle';
    },
  },
});
