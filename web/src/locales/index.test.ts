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

    expect(i18n.global.t('user.userList.listTitle')).toBe('用户管理');
    expect(i18n.global.t('rbac.roleList.listTitle')).toBe('角色管理');
    expect(i18n.global.t('accessControl.overview.title')).toBe('访问控制概览');
  });

  it('deep merges nested locale namespaces instead of replacing the whole top-level branch', async () => {
    const { mergeLocaleMessages } = await import('./index');

    expect(
      mergeLocaleMessages(
        {
          menu: {
            user: {
              title: '用户管理',
            },
          },
        },
        {
          menu: {
            role: {
              title: '角色管理',
            },
          },
        },
      ),
    ).toEqual({
      menu: {
        user: {
          title: '用户管理',
        },
        role: {
          title: '角色管理',
        },
      },
    });
  });
});
