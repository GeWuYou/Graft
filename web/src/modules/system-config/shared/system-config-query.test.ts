import { beforeEach, describe, expect, it, vi } from 'vitest';

import { queryClient } from '@/shared/query';

import { getSystemConfigs } from '../api/system-config';
import type { SystemConfigItem } from '../types/system-config';
import { systemConfigQueryKeys, upsertSystemConfigCache } from './system-config-query';

vi.mock('../api/system-config', () => ({
  getSystemConfigs: vi.fn(),
}));

describe('system config query cache', () => {
  beforeEach(() => {
    queryClient.clear();
    vi.mocked(getSystemConfigs).mockReset();
  });

  it('uses a stable module key and replaces a mutation result in the cached collection', () => {
    const previous = systemConfigItem({ effective_value: '7' });
    queryClient.setQueryData(systemConfigQueryKeys.list(), {
      items: [previous],
      total: 1,
    });

    upsertSystemConfigCache(systemConfigItem({ effective_value: '14' }));

    expect(queryClient.getQueryData(systemConfigQueryKeys.list())).toMatchObject({
      items: [expect.objectContaining({ key: 'logger.retention', effective_value: '14' })],
    });
    expect(getSystemConfigs).not.toHaveBeenCalled();
  });
});

function systemConfigItem(overrides: Partial<SystemConfigItem> = {}): SystemConfigItem {
  return {
    key: 'logger.retention',
    module: 'core.logger',
    group: 'retention',
    type: 'string',
    config_schema: {},
    default_value: '30',
    effective_value: '30',
    has_override: false,
    masked: false,
    sensitive: false,
    runtime_apply_mode: 'runtime_hot',
    restart_required: false,
    status: 'default',
    ...overrides,
  };
}
