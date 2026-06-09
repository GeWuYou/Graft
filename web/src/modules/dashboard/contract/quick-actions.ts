// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';

import type { DashboardQuickLink } from '../types/dashboard';

type SystemConfigItem = components['schemas']['system-config-item'];

const DASHBOARD_QUICK_ACTION_CONFIG_KEY = {
  ENABLED: 'dashboard.quick_actions.enabled',
  MAX_ITEMS: 'dashboard.quick_actions.max_items',
  STRATEGY: 'dashboard.quick_actions.strategy',
} as const;

export const DASHBOARD_QUICK_ACTION_STORAGE_KEY = {
  ROUTE_USAGE: 'dashboard:quick-actions:route-usage',
} as const;

export const DASHBOARD_QUICK_ACTION_STRATEGY = {
  MOST_USED: 'most_used',
  RECENT: 'recent',
  HYBRID: 'hybrid',
} as const;

export type DashboardQuickActionStrategy =
  (typeof DASHBOARD_QUICK_ACTION_STRATEGY)[keyof typeof DASHBOARD_QUICK_ACTION_STRATEGY];

export type DashboardQuickActionConfig = {
  enabled: boolean;
  maxItems: number;
  strategy: DashboardQuickActionStrategy;
};

export type DashboardQuickActionUsageRecord = {
  accessCount: number;
  lastAccessAt: string;
};

export type DashboardQuickActionUsageMap = Record<string, DashboardQuickActionUsageRecord>;

export type DashboardQuickActionViewModel = DashboardQuickLink & {
  accessCount: number;
  lastAccessAt: string;
  pinned: boolean;
};

export type DashboardQuickActionConfigParseDiagnostic = {
  key: string;
  error: unknown;
};

export type ResolveDashboardQuickActionConfigOptions = {
  onInvalidConfigValue?: (diagnostic: DashboardQuickActionConfigParseDiagnostic) => void;
};

export const DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG: DashboardQuickActionConfig = {
  enabled: true,
  maxItems: 8,
  strategy: DASHBOARD_QUICK_ACTION_STRATEGY.HYBRID,
};

const strategyValues = new Set<string>(Object.values(DASHBOARD_QUICK_ACTION_STRATEGY));

function isDashboardQuickActionStrategy(value: unknown): value is DashboardQuickActionStrategy {
  return typeof value === 'string' && strategyValues.has(value);
}

export function resolveDashboardQuickActionConfig(
  items: SystemConfigItem[],
  options: ResolveDashboardQuickActionConfigOptions = {},
) {
  const config = { ...DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG };

  for (const item of items) {
    const value = parseSystemConfigValue(item.key, item.effective_value, options);
    switch (item.key) {
      case DASHBOARD_QUICK_ACTION_CONFIG_KEY.ENABLED:
        if (typeof value === 'boolean') {
          config.enabled = value;
        }
        break;
      case DASHBOARD_QUICK_ACTION_CONFIG_KEY.MAX_ITEMS:
        if (typeof value === 'number' && Number.isInteger(value) && value > 0) {
          config.maxItems = Math.min(value, 24);
        }
        break;
      case DASHBOARD_QUICK_ACTION_CONFIG_KEY.STRATEGY:
        if (isDashboardQuickActionStrategy(value)) {
          config.strategy = value;
        }
        break;
      default:
        break;
    }
  }

  return config;
}

function parseSystemConfigValue(
  key: string,
  value: string | null | undefined,
  options: ResolveDashboardQuickActionConfigOptions,
) {
  if (!value?.trim()) {
    return undefined;
  }

  try {
    return JSON.parse(value) as unknown;
  } catch (error) {
    options.onInvalidConfigValue?.({ key, error });
    return undefined;
  }
}
