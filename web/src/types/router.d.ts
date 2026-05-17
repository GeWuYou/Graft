import 'vue-router';

import type { Component, DefineComponent, FunctionalComponent } from 'vue';

import type { LocalizedTitle } from '@/contracts/i18n/locales';

export {};
declare module 'vue-router' {
  interface RouteMeta {
    title?: LocalizedTitle;
    titleKey?: string;
    icon?: string | Component | FunctionalComponent | DefineComponent;
    expanded?: boolean;
    orderNo?: number;
    hidden?: boolean;
    hiddenBreadcrumb?: boolean;
    single?: boolean;
    keepAlive?: boolean;
    frameSrc?: string;
    frameBlank?: boolean;
  }
}
