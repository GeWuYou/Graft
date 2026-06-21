import type { LocalizedTitle, SupportedLocale } from '@/contracts/i18n/locales';
import { LOCALE } from '@/contracts/i18n/locales';
import type { AppRouteMeta, AppRoutePageSurface } from '@/utils/types';

export type RouteTitleSlot = 'page' | 'tab' | 'breadcrumb';
export type PageSurfaceType = AppRoutePageSurface;
type RouteMetaTitleInput = Omit<Partial<AppRouteMeta>, 'title' | 'semanticTitle' | 'breadcrumbTitle' | 'tabTitle'> & {
  title?: LocalizedTitle | string;
  semanticTitle?: LocalizedTitle | string;
  breadcrumbTitle?: LocalizedTitle | string;
  tabTitle?: LocalizedTitle | string;
};

export function toLocalizedTitle(title?: LocalizedTitle | string): LocalizedTitle | undefined {
  if (!title) {
    return undefined;
  }

  if (typeof title === 'string') {
    return {
      [LOCALE.ZH_CN]: title,
      [LOCALE.EN_US]: title,
    };
  }

  return title;
}

export function resolveRouteLocalizedTitle(meta?: RouteMetaTitleInput, slot: RouteTitleSlot = 'page') {
  if (!meta) {
    return undefined;
  }

  if (slot === 'tab') {
    return meta.tabTitle ?? meta.semanticTitle ?? meta.title;
  }

  if (slot === 'breadcrumb') {
    return meta.breadcrumbTitle ?? meta.semanticTitle ?? meta.title;
  }

  return meta.semanticTitle ?? meta.title;
}

export function renderLocalizedTitle(
  title: LocalizedTitle | string | undefined,
  locale: SupportedLocale,
  fallback = '',
) {
  const localizedTitle = toLocalizedTitle(title);
  if (!localizedTitle) {
    return fallback;
  }

  return localizedTitle[locale] || localizedTitle[LOCALE.ZH_CN] || localizedTitle[LOCALE.EN_US] || fallback;
}

export function resolvePageSurfaceType(meta?: RouteMetaTitleInput): PageSurfaceType {
  if (meta?.pageSurface) {
    return meta.pageSurface;
  }

  if (meta?.dashboard || meta?.pageKind === 'overview') {
    return 'overview-dashboard';
  }

  if (meta?.pageKind === 'list') {
    return 'paged-table';
  }

  if (meta?.pageKind && ['detail', 'investigation'].includes(meta.pageKind)) {
    return 'form-detail';
  }

  return 'shell';
}
