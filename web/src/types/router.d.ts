import 'vue-router';

import type { Component, DefineComponent, FunctionalComponent } from 'vue';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import type { AppRoutePageKind, GovernanceDomain, PageFooterMeta } from '@/utils/types';

export {};
declare module 'vue-router' {
  interface RouteMeta {
    title?: LocalizedTitle;
    titleKey?: string;
    domain?: GovernanceDomain;
    domainTitle?: LocalizedTitle;
    semanticTitle?: LocalizedTitle;
    breadcrumbTitle?: LocalizedTitle;
    tabTitle?: LocalizedTitle;
    tabGroup?: string;
    dashboard?: boolean;
    pageKind?: AppRoutePageKind;
    investigationSurface?: boolean;
    icon?: string | Component | FunctionalComponent | DefineComponent;
    expanded?: boolean;
    orderNo?: number;
    hidden?: boolean;
    hiddenMenu?: boolean;
    hiddenBreadcrumb?: boolean;
    single?: boolean;
    keepAlive?: boolean;
    frameSrc?: string;
    frameBlank?: boolean;
    footer?: false | PageFooterMeta;
  }
}
