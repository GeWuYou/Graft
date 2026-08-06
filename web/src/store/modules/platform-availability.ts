import { defineStore } from 'pinia';

import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

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
    recordFailure(path?: string) {
      this.consecutiveFailures += 1;
      if (path && path !== '/result/service-unavailable') {
        this.pendingPath = path;
      }
      if (this.consecutiveFailures >= FAILURE_THRESHOLD) {
        this.status = 'unavailable';
      }
    },
    recordSuccess() {
      this.consecutiveFailures = 0;
      this.lastCheckedAt = Date.now();
      this.status = 'healthy';
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
      this.probePromise = request
        .get<{ status: string }>({ url: OPENAPI_RUNTIME_PATH.getHealthz, _skipAuthRefresh: true })
        .then(() => {
          this.recordSuccess();
          return true;
        })
        .catch(() => {
          this.recordFailure();
          // healthz 是直接的控制面探测；与业务请求候选信号不同，单次失败即可接管页面。
          this.status = 'unavailable';
          return false;
        })
        .finally(() => {
          this.probePromise = null;
        });
      return this.probePromise;
    },
  },
});
