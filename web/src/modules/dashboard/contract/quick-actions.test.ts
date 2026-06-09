// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG, resolveDashboardQuickActionConfig } from './quick-actions';

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
    restart_required: false,
    sensitive: false,
    status: 'default',
    type: 'string',
  } as const;
}

describe('dashboard quick-action contract helpers', () => {
  it('reports invalid system-config JSON with the config key context', () => {
    const onInvalidConfigValue = vi.fn();

    const config = resolveDashboardQuickActionConfig([systemConfigItem('dashboard.quick_actions.max_items', '{')], {
      onInvalidConfigValue,
    });

    expect(config).toEqual(DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG);
    expect(onInvalidConfigValue).toHaveBeenCalledWith({
      key: 'dashboard.quick_actions.max_items',
      error: expect.any(SyntaxError),
    });
  });
});
