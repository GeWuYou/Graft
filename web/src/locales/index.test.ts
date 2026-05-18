import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { STORAGE_KEY } from '@/contracts/storage/keys';

describe('locales bootstrap', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('persists the default canonical locale on startup when no stored locale exists', async () => {
    await import('./index');

    expect(localStorage.getItem(STORAGE_KEY.LOCALE)).toBe('zh-CN');
  });

  it('normalizes legacy stored locale values on startup', async () => {
    localStorage.setItem(STORAGE_KEY.LOCALE, 'en_US');

    await import('./index');

    expect(localStorage.getItem(STORAGE_KEY.LOCALE)).toBe('en-US');
  });

  it('merges module-owned locale catalogs into the app i18n registry', async () => {
    const { i18n } = await import('./index');

    expect(i18n.global.t('user.userList.listTitle')).toBe('用户列表');
    expect(i18n.global.t('rbac.roleList.listTitle')).toBe('角色概览');
  });
});
