import { defineStore } from 'pinia';

import { setPlatformQueryOnline } from '@/shared/query/client';
import { setRealtimePlatformAvailable } from '@/shared/realtime/platform-availability';
import { probePlatformHealth, registerPlatformAvailabilityBridge } from '@/utils/request';

/** 浏览器控制面可达性的壳层状态，不表达任一模块或资源的健康度。 */
export type PlatformAvailabilityStatus = 'unknown' | 'healthy' | 'degraded' | 'unavailable' | 'recovering';

type PlatformAvailabilityState = {
  lastCheckedAt: number | null;
  status: PlatformAvailabilityStatus;
  consecutiveFailures: number;
  pendingPath: string | null;
  probePromise: Promise<boolean> | null;
};

const FAILURE_THRESHOLD = 2;
const RECOVERY_PROBE_DELAY_MS = 3_000;
const HEALTHY_PROBE_DELAY_MS = 10_000;
const HEALTH_PROBE_TIMEOUT_MS = 5_000;
let recoveryProbeTimer: ReturnType<typeof setTimeout> | null = null;

function clearRecoveryProbe() {
  if (recoveryProbeTimer === null) {
    return;
  }
  clearTimeout(recoveryProbeTimer);
  recoveryProbeTimer = null;
}

function initialState(): PlatformAvailabilityState {
  return {
    lastCheckedAt: null,
    status: 'unknown',
    consecutiveFailures: 0,
    pendingPath: null,
    probePromise: null,
  };
}

/** 平台可达性唯一前端 authority；Query 与业务模块只能消费其状态。 */
export const usePlatformAvailabilityStore = defineStore('platform-availability', {
  state: initialState,
  getters: {
    isUnavailable: (state) => state.status === 'unavailable',
    allowsBusinessTraffic: (state) => state.status !== 'unavailable',
  },
  actions: {
    bindRequestBridge() {
      registerPlatformAvailabilityBridge({
        allowsBusinessTraffic: () => this.allowsBusinessTraffic,
        reportTransportFailure: () => this.recordFailure(),
      });
    },
    recordFailure() {
      this.consecutiveFailures += 1;
      if (this.consecutiveFailures >= FAILURE_THRESHOLD) {
        this.enterUnavailable();
      }
    },
    recordSuccess() {
      clearRecoveryProbe();
      this.consecutiveFailures = 0;
      this.lastCheckedAt = Date.now();
      this.status = 'healthy';
      setPlatformQueryOnline(true);
      setRealtimePlatformAvailable(true);
      this.scheduleHealthProbe(HEALTHY_PROBE_DELAY_MS);
    },
    enterUnavailable() {
      this.status = 'unavailable';
      setPlatformQueryOnline(false);
      setRealtimePlatformAvailable(false);
      this.scheduleHealthProbe(RECOVERY_PROBE_DELAY_MS);
    },
    scheduleHealthProbe(delay: number) {
      if (recoveryProbeTimer !== null) {
        return;
      }
      recoveryProbeTimer = setTimeout(() => {
        recoveryProbeTimer = null;
        void this.checkHealth();
      }, delay);
    },
    stopHealthMonitoring() {
      clearRecoveryProbe();
    },
    beginRecovery() {
      this.status = 'recovering';
    },
    consumePendingPath(fallback = '/') {
      const path = this.pendingPath || fallback;
      this.pendingPath = null;
      return path;
    },
    async checkHealth(): Promise<boolean> {
      if (this.probePromise) {
        return this.probePromise;
      }
      this.beginRecovery();
      const controller = new AbortController();
      const timeoutID = setTimeout(() => controller.abort(), HEALTH_PROBE_TIMEOUT_MS);
      this.probePromise = probePlatformHealth(controller.signal)
        .then(() => {
          this.recordSuccess();
          return true;
        })
        .catch(() => {
          this.recordFailure();
          // healthz 是直接的控制面探测；与业务请求候选信号不同，单次失败即可接管页面。
          this.enterUnavailable();
          return false;
        })
        .finally(() => {
          clearTimeout(timeoutID);
          this.probePromise = null;
        });
      return this.probePromise;
    },
  },
});
