import type { components } from '@/contracts/openapi/generated/schema';

import type { DashboardQuickActionLink } from './quick-action-links';

type SystemConfigItem = components['schemas']['system-config-item'];

export const DASHBOARD_QUICK_ACTION_CONFIG_KEY = 'dashboard.quick_actions';

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

export type DashboardQuickActionViewModel = DashboardQuickActionLink & {
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
  maxItems: 4,
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
    if (item.key !== DASHBOARD_QUICK_ACTION_CONFIG_KEY) {
      continue;
    }
    const value = parseSystemConfigValue(item.key, item.effective_value, options);
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      continue;
    }

    const partial = value as Partial<Record<keyof DashboardQuickActionConfig, unknown>>;
    if (typeof partial.enabled === 'boolean') {
      config.enabled = partial.enabled;
    }
    if (typeof partial.maxItems === 'number' && Number.isInteger(partial.maxItems) && partial.maxItems > 0) {
      config.maxItems = Math.min(partial.maxItems, 24);
    }
    if (isDashboardQuickActionStrategy(partial.strategy)) {
      config.strategy = partial.strategy;
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
