import { describe, expect, it, vi } from 'vitest';

import {
  DASHBOARD_QUICK_ACTION_CONFIG_KEY,
  DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG,
  resolveDashboardQuickActionConfig,
} from './quick-actions';

function systemConfigItem(key: string, effectiveValue: string) {
  return {
    config_schema: {},
    default_value: null,
    effective_value: effectiveValue,
    group: 'dashboard.quick_actions',
    has_override: false,
    key,
    masked: false,
    module: 'core',
    runtime_apply_mode: 'unknown',
    restart_required: false,
    sensitive: false,
    status: 'default',
    type: 'string',
  } as const;
}

describe('dashboard quick-action contract helpers', () => {
  it('keeps the fallback maximum aligned with one desktop quick-action row', () => {
    expect(DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG.maxItems).toBe(4);
  });

  it('reports invalid system-config JSON with the config key context', () => {
    const onInvalidConfigValue = vi.fn();

    const config = resolveDashboardQuickActionConfig([systemConfigItem(DASHBOARD_QUICK_ACTION_CONFIG_KEY, '{')], {
      onInvalidConfigValue,
    });

    expect(config).toEqual(DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG);
    expect(onInvalidConfigValue).toHaveBeenCalledWith({
      key: DASHBOARD_QUICK_ACTION_CONFIG_KEY,
      error: expect.any(SyntaxError),
    });
  });

  it('reads the canonical quick-actions object config', () => {
    const config = resolveDashboardQuickActionConfig([
      systemConfigItem(DASHBOARD_QUICK_ACTION_CONFIG_KEY, '{"enabled":false,"maxItems":2,"strategy":"most_used"}'),
    ]);

    expect(config).toEqual({ enabled: false, maxItems: 2, strategy: 'most_used' });
  });

  it('does not consume removed flat quick-action keys', () => {
    const config = resolveDashboardQuickActionConfig([
      systemConfigItem('dashboard.quick_actions.enabled', 'false'),
      systemConfigItem('dashboard.quick_actions.max_items', '1'),
      systemConfigItem('dashboard.quick_actions.strategy', '"most_used"'),
    ]);

    expect(config).toEqual(DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG);
  });
});
