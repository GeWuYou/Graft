import { defineStore } from 'pinia';

export type RealtimeFreezeReason = 'shell-sidebar-motion';
export type RealtimeSchedulerPhase = 'running' | 'freezing' | 'resuming';
export type RealtimeFreezeToken = number;

type RealtimeSchedulerState = {
  activeFreezeReasonsByToken: Record<number, RealtimeFreezeReason>;
  nextTokenId: number;
  phase: RealtimeSchedulerPhase;
  resumeFrameHandle: number | null;
};

function createInitialRealtimeSchedulerState(): RealtimeSchedulerState {
  return {
    activeFreezeReasonsByToken: {},
    nextTokenId: 0,
    phase: 'running',
    resumeFrameHandle: null,
  };
}

export const useRealtimeSchedulerStore = defineStore('realtime-scheduler', {
  state: createInitialRealtimeSchedulerState,
  getters: {
    activeFreezeCount: (state) => Object.keys(state.activeFreezeReasonsByToken).length,
    isFrozen(): boolean {
      return this.phase !== 'running';
    },
    allowPolling(): boolean {
      return !this.isFrozen;
    },
    allowSnapshotCommit(): boolean {
      return !this.isFrozen;
    },
  },
  actions: {
    freeze(reason: RealtimeFreezeReason): RealtimeFreezeToken {
      this.cancelResumeTransition();
      const token = ++this.nextTokenId;
      this.activeFreezeReasonsByToken[token] = reason;
      if (this.phase !== 'freezing') {
        this.phase = 'freezing';
      }
      return token;
    },
    release(token: RealtimeFreezeToken) {
      if (!(token in this.activeFreezeReasonsByToken)) {
        return;
      }
      delete this.activeFreezeReasonsByToken[token];
      if (this.activeFreezeCount > 0) {
        return;
      }
      this.beginResumeTransition();
    },
    reset() {
      this.cancelResumeTransition();
      this.activeFreezeReasonsByToken = {};
      this.phase = 'running';
      this.nextTokenId = 0;
    },
    beginResumeTransition() {
      this.cancelResumeTransition();
      this.phase = 'resuming';
      if (typeof window === 'undefined') {
        this.finishResumeTransition();
        return;
      }
      this.resumeFrameHandle = window.requestAnimationFrame(() => {
        this.resumeFrameHandle = null;
        this.finishResumeTransition();
      });
    },
    finishResumeTransition() {
      if (this.activeFreezeCount > 0) {
        this.phase = 'freezing';
        return;
      }
      this.phase = 'running';
    },
    cancelResumeTransition() {
      if (this.resumeFrameHandle !== null && typeof window !== 'undefined') {
        window.cancelAnimationFrame(this.resumeFrameHandle);
      }
      this.resumeFrameHandle = null;
    },
  },
});
