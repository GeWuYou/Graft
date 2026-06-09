// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';

import { createLogger } from '@/utils/logger';

import {
  DASHBOARD_QUICK_ACTION_STORAGE_KEY,
  DASHBOARD_QUICK_ACTION_STRATEGY,
  type DashboardQuickActionConfig,
  type DashboardQuickActionUsageMap,
  type DashboardQuickActionUsageRecord,
  type DashboardQuickActionViewModel,
} from '../contract/quick-actions';
import type { DashboardQuickLink } from '../types/dashboard';

const INVALID_LAST_ACCESS_TIME = 0;
const logger = createLogger('dashboard.quickActions');

function canUseLocalStorage() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function readUsageMap(): DashboardQuickActionUsageMap {
  if (!canUseLocalStorage()) {
    return {};
  }

  try {
    const parsed = JSON.parse(
      window.localStorage.getItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE) || '{}',
    ) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      logger.warn('dashboard quick-action usage storage payload invalid', {
        storageKey: DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE,
      });
      return {};
    }

    return Object.fromEntries(
      Object.entries(parsed).flatMap(([route, value]) => {
        const record = normalizeUsageRecord(value);
        return record ? [[route, record]] : [];
      }),
    );
  } catch (error) {
    logger.warn('dashboard quick-action usage storage parse failed', {
      storageKey: DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE,
      error,
    });
    return {};
  }
}

function normalizeUsageRecord(value: unknown): DashboardQuickActionUsageRecord | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }

  const record = value as Partial<DashboardQuickActionUsageRecord>;
  const accessCount = Number(record.accessCount);
  const lastAccessAt = normalizeLastAccessAt(record.lastAccessAt);
  if (!Number.isFinite(accessCount) || accessCount < 0) {
    return null;
  }

  return {
    accessCount,
    lastAccessAt,
  };
}

function normalizeLastAccessAt(value: unknown) {
  if (typeof value !== 'string' || !value.trim()) {
    return '';
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? value : '';
}

function writeUsageMap(value: DashboardQuickActionUsageMap) {
  if (!canUseLocalStorage()) {
    return;
  }

  window.localStorage.setItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE, JSON.stringify(value));
}

function lastAccessTime(value: string) {
  if (!value.trim()) {
    return INVALID_LAST_ACCESS_TIME;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : INVALID_LAST_ACCESS_TIME;
}

function score(
  link: DashboardQuickActionViewModel,
  config: DashboardQuickActionConfig,
  maxAccessCount: number,
  maxRecentTime: number,
) {
  if (link.pinned) {
    return Number.POSITIVE_INFINITY;
  }

  const normalizedAccess = maxAccessCount > 0 ? link.accessCount / maxAccessCount : 0;
  const recentTime = lastAccessTime(link.lastAccessAt);
  const normalizedRecent = maxRecentTime > 0 ? recentTime / maxRecentTime : 0;
  if (config.strategy === DASHBOARD_QUICK_ACTION_STRATEGY.MOST_USED) {
    return normalizedAccess;
  }
  if (config.strategy === DASHBOARD_QUICK_ACTION_STRATEGY.RECENT) {
    return normalizedRecent;
  }

  return normalizedAccess * 0.7 + normalizedRecent * 0.3;
}

export function useDashboardQuickActions(links: () => DashboardQuickLink[], config: () => DashboardQuickActionConfig) {
  const usage = ref<DashboardQuickActionUsageMap>(readUsageMap());

  const rankedLinks = computed<DashboardQuickActionViewModel[]>(() => {
    const viewModels = links().map((link) => {
      const record = usage.value[link.route_location];
      return {
        ...link,
        accessCount: record?.accessCount ?? 0,
        lastAccessAt: record?.lastAccessAt ?? '',
        pinned: Boolean((link as Partial<DashboardQuickActionViewModel>).pinned),
      };
    });
    const maxAccessCount = Math.max(...viewModels.map((link) => link.accessCount), 0);
    const maxRecentTime = Math.max(...viewModels.map((link) => lastAccessTime(link.lastAccessAt)), 0);
    const currentConfig = config();

    return viewModels.sort((left, right) => {
      if (left.pinned !== right.pinned) {
        return left.pinned ? -1 : 1;
      }
      const scoreDelta =
        score(right, currentConfig, maxAccessCount, maxRecentTime) -
        score(left, currentConfig, maxAccessCount, maxRecentTime);
      if (scoreDelta !== 0) {
        return scoreDelta;
      }
      if (left.order !== right.order) {
        return left.order - right.order;
      }
      return left.id.localeCompare(right.id);
    });
  });

  function recordAccess(route: string) {
    const target = route.trim();
    if (!target) {
      return;
    }

    const current = usage.value[target];
    usage.value = {
      ...usage.value,
      [target]: {
        accessCount: (current?.accessCount ?? 0) + 1,
        lastAccessAt: new Date().toISOString(),
      },
    };
    writeUsageMap(usage.value);
  }

  return {
    rankedLinks,
    recordAccess,
  };
}
