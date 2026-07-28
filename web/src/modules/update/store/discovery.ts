import { defineStore } from 'pinia';

import { usePermissionStore } from '@/store';

import { checkForUpdates, getUpdateStatus } from '../api/update';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import type { UpdateStatus } from '../types/update';

export type UpdateDiscoveryPhase = 'idle' | 'loading' | 'ready' | 'error';

const previewRefreshTtlMs = 60_000;

/** 更新发现快照的模块级状态源，避免壳层入口和管理页分别请求同一份服务端事实。 */
export const useUpdateDiscoveryStore = defineStore('update-discovery', {
  state: () => ({
    status: null as UpdateStatus | null,
    phase: 'idle' as UpdateDiscoveryPhase,
    error: '',
    generation: 0,
    requestPromise: null as Promise<UpdateStatus | null> | null,
    previewRequestPromise: null as Promise<UpdateStatus> | null,
    lastRefreshAt: 0,
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
    async refreshSnapshot() {
      const generation = ++this.generation;
      this.requestPromise = null;
      this.phase = 'loading';
      this.error = '';
      try {
        const status = await checkForUpdates();
        if (generation === this.generation) {
          this.status = status;
          this.phase = 'ready';
          this.lastRefreshAt = Date.now();
        }
        return status;
      } catch (error) {
        if (generation === this.generation) {
          if (this.status) {
            this.status = { ...this.status, cache_stale: true, check_error: 'check-failed' };
          }
          this.phase = 'error';
          this.error = 'check-failed';
        }
        throw error;
      }
    },
    async refreshPreviewSnapshot() {
      // 预览入口共享请求 Promise；短 TTL 只复用没有 stale/error 标记的快照，实际刷新仍统一走 refreshSnapshot。
      if (this.previewRequestPromise) {
        return this.previewRequestPromise;
      }

      const hasFreshSnapshot =
        this.status &&
        !this.status.cache_stale &&
        !this.status.check_error &&
        this.lastRefreshAt > 0 &&
        Date.now() - this.lastRefreshAt < previewRefreshTtlMs;
      if (hasFreshSnapshot) {
        return this.status;
      }

      this.previewRequestPromise = this.refreshSnapshot().finally(() => {
        this.previewRequestPromise = null;
      });
      return this.previewRequestPromise;
    },
    async revalidateVisibleSnapshot() {
      const permissionStore = usePermissionStore();
      if (!permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ)) {
        return null;
      }
      const generation = this.generation;
      const status = await getUpdateStatus();
      if (generation !== this.generation) {
        return null;
      }
      this.replaceSnapshot(status);
      return status;
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
      this.previewRequestPromise = null;
      this.lastRefreshAt = 0;
    },
  },
});
