import { describe, expect, it } from 'vitest';

import { hasUnresolvedRouteTitleKey, isRouteTitleKey, localizeRouteTitle } from './title';

describe('localizeRouteTitle', () => {
  it('prefers the bootstrap title_key when the frontend locale catalog defines it', () => {
    expect(localizeRouteTitle('用户管理', 'menu.user_list.title')).toEqual({
      'zh-CN': '用户管理',
      'en-US': 'Users',
    });
    expect(localizeRouteTitle('可观测性', 'monitor.sectionTitle')).toEqual({
      'zh-CN': '可观测性',
      'en-US': 'Observability',
    });
    expect(localizeRouteTitle('', 'menu.monitor.serviceStatus.title')).toEqual({
      'zh-CN': '服务状态',
      'en-US': 'Service Status',
    });
    expect(localizeRouteTitle('', 'container.route.images.title')).toEqual({
      'zh-CN': '镜像',
      'en-US': 'Images',
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

  it('detects unresolved message keys inside navigation titles', () => {
    expect(
      hasUnresolvedRouteTitleKey(
        {
          'zh-CN': '基础设施 / Docker / container.route.images.title',
          'en-US': 'Infrastructure / Docker / container.route.images.title',
        },
        'container.route.images.title',
      ),
    ).toBe(true);
    expect(
      hasUnresolvedRouteTitleKey({
        'zh-CN': '基础设施 / Docker / 镜像',
        'en-US': 'Infrastructure / Docker / Images',
      }),
    ).toBe(false);
  });

  it('does not classify hostname-like product titles as message keys', () => {
    expect(isRouteTitleKey('docker.io')).toBe(false);
    expect(isRouteTitleKey('registry.example')).toBe(false);
    expect(
      hasUnresolvedRouteTitleKey({
        'zh-CN': 'docker.io',
        'en-US': 'docker.io',
      }),
    ).toBe(false);
    expect(
      hasUnresolvedRouteTitleKey({
        'zh-CN': 'registry.example',
        'en-US': 'release.channel.alpha',
      }),
    ).toBe(false);
    expect(isRouteTitleKey('monitor.sectionTitle')).toBe(true);
    expect(isRouteTitleKey('missing.route.title', 'missing.route.title')).toBe(true);
  });
});
