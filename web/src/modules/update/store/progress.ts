import { defineStore } from 'pinia';

import type { RealtimeTopicEventStreamController } from '@/shared/realtime/sse-client';
import { isApiRequestError } from '@/utils/request';

import {
  getActiveUpdateOperation,
  getUpdateOperation,
  getUpdateOperationDiagnostic,
  getUpdateOperationEvents,
  recoverUpdateOperation,
  subscribeToUpdateOperation,
} from '../api/update';
import type {
  UpdateFailureDiagnostic,
  UpdateOperation,
  UpdateOperationEvent,
  UpdateOperationLaunchAcknowledgement,
  UpdateOperationRealtimeMessage,
} from '../types/update';

type ProgressPhase = 'idle' | 'running' | 'reconnecting' | 'success' | 'failed' | 'unavailable';

const SESSION_STORAGE_KEY = 'graft.platform-update.operation-id';
const terminalSuccess = new Set<UpdateOperation['phase']>(['SUCCESS']);
const terminalFailure = new Set<UpdateOperation['phase']>(['FAILED', 'ROLLBACK']);
const POLL_INTERVAL_MS = 3000;
const MAX_SNAPSHOT_RETRIES = 5;
const runnerDisconnectedStateSources = new Set<UpdateOperation['state_source']>(['runner_lost', 'runner_terminated']);

function isRunnerDisconnected(operation: UpdateOperation | null | undefined) {
  return Boolean(operation && runnerDisconnectedStateSources.has(operation.state_source));
}

function isUnrecoverableSnapshotError(error: unknown) {
  return isApiRequestError(error) && error.status >= 400 && error.status < 500;
}

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

function isOperationEvent(
  message: UpdateOperationRealtimeMessage,
): message is Extract<UpdateOperationRealtimeMessage, { event: UpdateOperationEvent }> {
  return 'event' in message;
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
      snapshotRetryCount: 0,
      events: [] as UpdateOperationEvent[],
      failureDiagnostic: null as UpdateFailureDiagnostic | null,
      failureDiagnosticLoading: false,
      failureDiagnosticError: false,
      latestEventRevision: 0,
      recoveringActiveOperation: false,
      recoveryLoading: false,
      recoveryError: false,
      recoveryPending: false,
    };
  },
  getters: {
    visible: (state) => state.phase !== 'idle',
  },
  actions: {
    async begin(acknowledgement: UpdateOperationLaunchAcknowledgement, recoveryPending = false) {
      this.session += 1;
      this.stopStream();
      this.stopPolling();
      persistOperation(acknowledgement.operation_id);
      this.operationID = acknowledgement.operation_id;
      this.operation = null;
      this.lastActivePhase = null;
      this.snapshotRetryCount = 0;
      this.events = [];
      this.failureDiagnostic = null;
      this.failureDiagnosticLoading = false;
      this.failureDiagnosticError = false;
      this.latestEventRevision = 0;
      this.recoveryLoading = false;
      this.recoveryError = false;
      this.recoveryPending = recoveryPending;
      this.phase = 'reconnecting';
      await this.refreshSnapshot(this.session);
    },
    async resume() {
      if (this.operationID && !this.stream && this.phase !== 'idle' && !this.isTerminal()) {
        await this.refreshSnapshot(this.session);
        return;
      }
      if (this.operationID || this.recoveringActiveOperation || this.phase !== 'idle') return;
      this.recoveringActiveOperation = true;
      try {
        const operation = await getActiveUpdateOperation();
        if (!operation || this.operationID || this.phase !== 'idle') return;
        this.session += 1;
        const session = this.session;
        persistOperation(operation.operation_id);
        this.operationID = operation.operation_id;
        this.phase = 'reconnecting';
        await this.applyOperation(session, operation);
        if (!this.isTerminal()) await this.refreshEvents(session);
        if (!this.isTerminal() && !this.stream) this.connect(session);
      } catch {
        // 未知是否存在升级时，不能仅因 active 查询短暂失败而阻断后台壳；已知操作仍由快照恢复路径明确报告不可用。
      } finally {
        this.recoveringActiveOperation = false;
      }
    },
    async refreshSnapshot(session: number) {
      const operationID = this.operationID;
      if (!operationID || session !== this.session) return;
      try {
        const operation = await getUpdateOperation(operationID);
        this.snapshotRetryCount = 0;
        await this.applyOperation(session, operation);
        if (!this.isTerminal()) await this.refreshEvents(session);
        if (session === this.session && !this.isTerminal() && !this.stream) this.connect(session);
      } catch (error) {
        if (session !== this.session || this.isTerminal()) return;
        this.snapshotRetryCount += 1;
        if (isUnrecoverableSnapshotError(error) || this.snapshotRetryCount >= MAX_SNAPSHOT_RETRIES) {
          this.failSnapshotRecovery();
          return;
        }
        this.phase = 'reconnecting';
        this.startPolling(session);
      }
    },
    connect(session: number) {
      const operationID = this.operationID;
      if (!operationID || session !== this.session) return;
      this.stream = subscribeToUpdateOperation(operationID, {
        onOperation: async (message) => this.applyRealtimeMessage(session, message),
        onStateChange: (state) => {
          if (session !== this.session || this.isTerminal()) return;
          if (state === 'open') {
            void this.refreshEvents(session).finally(() => {
              if (session === this.session && !this.isTerminal()) {
                if (this.recoveryPending && isRunnerDisconnected(this.operation)) {
                  this.phase = 'reconnecting';
                  this.startPolling(session);
                  return;
                }
                this.phase = 'running';
                this.stopPolling();
              }
            });
          } else if (state !== 'idle') {
            this.phase = 'reconnecting';
            this.startPolling(session);
          }
        },
      });
    },
    async applyRealtimeMessage(session: number, message: UpdateOperationRealtimeMessage) {
      if (isOperationEvent(message)) {
        this.appendEvent(message.event);
        if (message.operation) await this.applyOperation(session, message.operation);
        return;
      }
      await this.applyOperation(session, message);
    },
    async refreshEvents(session: number) {
      const operationID = this.operationID;
      if (!operationID || session !== this.session) return;
      try {
        const events = await getUpdateOperationEvents(operationID, this.latestEventRevision);
        if (session !== this.session) return;
        events.forEach((event) => this.appendEvent(event));
      } catch (error) {
        if (isUnrecoverableSnapshotError(error)) this.failSnapshotRecovery();
      }
    },
    appendEvent(event: UpdateOperationEvent) {
      const trackedOperationID = this.operationID ?? this.operation?.operation_id;
      if (
        !trackedOperationID ||
        event.operation_id !== trackedOperationID ||
        event.revision <= this.latestEventRevision
      ) {
        return;
      }
      this.events.push(event);
      this.latestEventRevision = event.revision;
    },
    async applyOperation(session: number, operation: UpdateOperation) {
      if (session !== this.session) return;
      if (this.operationID && operation.operation_id !== this.operationID) return;
      this.operation = operation;
      this.operationID = operation.operation_id;
      if (isRunnerDisconnected(operation)) {
        if (this.recoveryPending) {
          this.phase = 'reconnecting';
          this.startPolling(session);
          return;
        }
        const terminalSession = ++this.session;
        this.phase = 'failed';
        this.lastActivePhase = null;
        this.stopStream();
        this.stopPolling();
        persistOperation(null);
        this.operationID = null;
        if (operation.failure_diagnostic_available !== false) {
          await this.loadFailureDiagnostic(terminalSession, operation.operation_id);
        } else {
          this.failureDiagnosticError = true;
        }
        return;
      }
      this.recoveryPending = false;
      if (!operation.state_available || operation.state_source === 'runner_state_unavailable') {
        this.failSnapshotRecovery();
        return;
      }
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
    async recoverTerminatedRunner() {
      const operation = this.operation;
      if (this.recoveryLoading || !operation || !isRunnerDisconnected(operation)) return;
      const recoverySession = this.session;
      const operationID = operation.operation_id;
      const isCurrentRecovery = () => this.session === recoverySession && this.operation?.operation_id === operationID;
      this.recoveryLoading = true;
      this.recoveryError = false;
      try {
        const acknowledgement = await recoverUpdateOperation(operationID);
        if (!isCurrentRecovery()) return;
        await this.begin(acknowledgement, true);
      } catch {
        if (isCurrentRecovery()) this.recoveryError = true;
      } finally {
        if (isCurrentRecovery()) this.recoveryLoading = false;
      }
    },
    isTerminal() {
      return this.phase === 'success' || this.phase === 'failed' || this.phase === 'unavailable';
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
    failSnapshotRecovery() {
      this.phase = 'unavailable';
      this.stopStream();
      this.stopPolling();
    },
    async loadFailureDiagnostic(session: number, operationID: string) {
      this.failureDiagnosticLoading = true;
      this.failureDiagnosticError = false;
      try {
        const diagnostic = await getUpdateOperationDiagnostic(operationID);
        if (session === this.session) this.failureDiagnostic = diagnostic;
      } catch {
        if (session === this.session) this.failureDiagnosticError = true;
      } finally {
        if (session === this.session) this.failureDiagnosticLoading = false;
      }
    },
    reset() {
      this.session += 1;
      this.stopStream();
      this.stopPolling();
      persistOperation(null);
      this.operation = null;
      this.operationID = null;
      this.lastActivePhase = null;
      this.snapshotRetryCount = 0;
      this.events = [];
      this.failureDiagnostic = null;
      this.failureDiagnosticLoading = false;
      this.failureDiagnosticError = false;
      this.latestEventRevision = 0;
      this.recoveringActiveOperation = false;
      this.recoveryLoading = false;
      this.recoveryError = false;
      this.recoveryPending = false;
      this.phase = 'idle';
    },
  },
});
