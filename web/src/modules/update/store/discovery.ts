import { defineStore } from 'pinia';

import { usePermissionStore } from '@/store';

import { getUpdateStatus } from '../api/update';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import type { UpdateStatus } from '../types/update';

export type UpdateDiscoveryPhase = 'idle' | 'loading' | 'ready' | 'error';

/** 更新发现快照的模块级状态源，避免壳层入口和管理页分别请求同一份服务端事实。 */
export const useUpdateDiscoveryStore = defineStore('update-discovery', {
  state: () => ({
    status: null as UpdateStatus | null,
    phase: 'idle' as UpdateDiscoveryPhase,
    error: '',
    generation: 0,
    requestPromise: null as Promise<UpdateStatus | null> | null,
  }),
  getters: {
    hasUpdate: (state) => Boolean(state.status?.latest && !state.status.cache_stale && !state.status.check_error),
  },
  actions: {
    async ensureSnapshot() {
      const permissionStore = usePermissionStore();
      if (!permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ)) {
        return null;
      }
      if (this.phase === 'ready') {
        return this.status;
      }
      if (this.requestPromise) {
        return this.requestPromise;
      }

      const generation = this.generation;
      this.phase = 'loading';
      this.requestPromise = getUpdateStatus()
        .then((status) => {
          if (generation === this.generation) {
            this.status = status;
            this.phase = 'ready';
            this.error = '';
          }
          return status;
        })
        .catch(() => {
          if (generation === this.generation) {
            this.phase = 'error';
            this.error = 'load-failed';
          }
          return null;
        })
        .finally(() => {
          if (generation === this.generation) {
            this.requestPromise = null;
          }
        });
      return this.requestPromise;
    },
    replaceSnapshot(status: UpdateStatus) {
      this.generation += 1;
      this.requestPromise = null;
      this.status = status;
      this.phase = 'ready';
      this.error = '';
    },
    invalidateSnapshot(error = 'check-failed') {
      this.generation += 1;
      this.requestPromise = null;
      if (this.status) {
        this.status = { ...this.status, cache_stale: true, check_error: error };
      }
      this.phase = 'error';
      this.error = error;
    },
    reset() {
      this.generation += 1;
      this.status = null;
      this.phase = 'idle';
      this.error = '';
      this.requestPromise = null;
    },
  },
});
