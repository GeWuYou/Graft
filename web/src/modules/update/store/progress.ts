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
    // HTTP 轮询不能撤销；每次会话切换递增该代号，使旧响应无法写回当前壳层状态。
    session: 0,
  }),
  getters: {
    visible: (state) => state.phase !== 'idle',
  },
  actions: {
    begin(operation: UpdateOperation) {
      this.session += 1;
      this.stopPolling();
      this.operation = operation;
      this.diagnostic = null;
      this.phase = 'running';
      void this.poll(this.session);
    },
    async poll(session: number) {
      if (session !== this.session) return;
      const operationID = this.operation?.operation_id;
      if (!operationID || this.phase === 'idle') return;
      try {
        const operation = await getUpdateOperation(operationID);
        if (session !== this.session) return;
        this.operation = operation;
        if (terminalSuccess.has(operation.status)) {
          this.phase = 'success';
          this.stopPolling();
          window.setTimeout(() => {
            if (session === this.session) window.location.reload();
          }, 1200);
          return;
        }
        if (terminalFailure.has(operation.status)) {
          this.phase = 'failed';
          this.stopPolling();
          try {
            const diagnostic = await getUpdateOperationDiagnostic(operationID);
            if (session !== this.session) return;
            this.diagnostic = diagnostic;
          } catch {
            if (session !== this.session) return;
            this.diagnostic = null;
          }
          return;
        }
        this.phase = 'running';
      } catch {
        if (session !== this.session) return;
        // Compose 重建会短暂中断 API；保留遮罩并继续轮询。
        this.phase = 'reconnecting';
      }
      if (session !== this.session) return;
      this.pollTimer = window.setTimeout(() => void this.poll(session), 2000);
    },
    stopPolling() {
      if (this.pollTimer !== null) window.clearTimeout(this.pollTimer);
      this.pollTimer = null;
    },
    reset() {
      this.session += 1;
      this.stopPolling();
      this.operation = null;
      this.diagnostic = null;
      this.phase = 'idle';
    },
  },
});
