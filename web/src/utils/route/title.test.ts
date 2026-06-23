import { describe, expect, it } from 'vitest';

import { localizeRouteTitle } from './title';

describe('localizeRouteTitle', () => {
  it('prefers the bootstrap title_key when the frontend locale catalog defines it', () => {
    expect(localizeRouteTitle('用户管理', 'menu.user_list.title')).toEqual({
      'zh-CN': '用户管理',
      'en-US': 'Users',
    });
    expect(localizeRouteTitle('服务管理', 'monitor.sectionTitle')).toEqual({
      'zh-CN': '服务管理',
      'en-US': 'Service Management',
    });
  });

  it('falls back to bootstrap title when title_key is missing or untranslated', () => {
    expect(localizeRouteTitle('角色管理')).toEqual({
      'zh-CN': '角色管理',
      'en-US': '角色管理',
    });
    expect(localizeRouteTitle('角色管理', 'menu.unknown.title')).toEqual({
      'zh-CN': '角色管理',
      'en-US': '角色管理',
    });
  });
});
