import { t } from '@/locales';

export function resolveDashboardText(key?: string, fallback = '') {
  if (!key) {
    return fallback;
  }

  const translated = t(key);
  return translated === key ? fallback || key : translated;
}
