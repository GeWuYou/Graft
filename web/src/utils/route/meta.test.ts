// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { LOCALE } from '@/contracts/i18n/locales';

import { renderLocalizedTitle, resolvePageSurfaceType, resolveRouteLocalizedTitle, toLocalizedTitle } from './meta';

describe('route meta helpers', () => {
  it('resolves domain-aware title chains for tabs and breadcrumbs', () => {
    const meta = {
      title: { 'zh-CN': '概览', 'en-US': 'Overview' },
      semanticTitle: { 'zh-CN': '安全审计 - 概览', 'en-US': 'Security Audit - Overview' },
      breadcrumbTitle: { 'zh-CN': '概览', 'en-US': 'Overview' },
      tabTitle: { 'zh-CN': '安全审计 - 概览', 'en-US': 'Security Audit - Overview' },
    };

    expect(resolveRouteLocalizedTitle(meta, 'page')).toEqual(meta.semanticTitle);
    expect(resolveRouteLocalizedTitle(meta, 'breadcrumb')).toEqual(meta.breadcrumbTitle);
    expect(resolveRouteLocalizedTitle(meta, 'tab')).toEqual(meta.tabTitle);
  });

  it('derives shell surface from meta instead of path prefixes', () => {
    expect(resolvePageSurfaceType({ dashboard: true, pageKind: 'overview' })).toBe('overview-dashboard');
    expect(resolvePageSurfaceType({ pageKind: 'list' })).toBe('list-form-detail');
    expect(resolvePageSurfaceType({ pageKind: 'investigation' })).toBe('list-form-detail');
    expect(resolvePageSurfaceType({ pageKind: 'runtime' })).toBe('shell');
  });

  it('normalizes string titles to localized payloads and renders locale fallbacks', () => {
    expect(toLocalizedTitle('概览')).toEqual({
      [LOCALE.ZH_CN]: '概览',
      [LOCALE.EN_US]: '概览',
    });
    expect(
      renderLocalizedTitle({ 'zh-CN': '安全审计 - 概览', 'en-US': 'Security Audit - Overview' }, LOCALE.EN_US),
    ).toBe('Security Audit - Overview');
  });
});
