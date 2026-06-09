import { describe, expect, it, vi } from 'vitest';

vi.mock('@/locales', () => ({
  t: (key: string) => {
    const translations: Record<string, string> = {
      'dashboard.known': '已翻译',
      'dashboard.widget.auditRiskEvents.highRisk.description': '过去 24 小时存在高风险事件',
    };
    return translations[key] ?? key;
  },
}));

import { hasDashboardTranslation, resolveDashboardRelatedText, resolveDashboardText } from './widget-i18n';

describe('dashboard widget i18n helpers', () => {
  it('prefers translated keys before server fallback text', () => {
    expect(resolveDashboardText('dashboard.known', 'Server fallback')).toBe('已翻译');
  });

  it('falls back to provided text only after detecting a missing key', () => {
    expect(resolveDashboardText('dashboard.missing', 'Server fallback')).toBe('Server fallback');
    expect(hasDashboardTranslation('dashboard.missing')).toBe(false);
  });

  it('uses a safe default when neither key nor fallback has display text', () => {
    expect(resolveDashboardText('dashboard.missing')).toBe('-');
    expect(resolveDashboardText(undefined, '   ', '默认')).toBe('默认');
  });

  it('resolves sibling description keys before accepting English payload fallbacks', () => {
    expect(
      resolveDashboardRelatedText(
        'dashboard.widget.auditRiskEvents.highRisk.title',
        'description',
        '1 high-risk events in the last 24 hours.',
      ),
    ).toBe('过去 24 小时存在高风险事件');
  });
});
