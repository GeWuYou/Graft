/* eslint-disable @typescript-eslint/no-unused-vars */
/// <reference types="vite/client" />

export {};

interface ImportMetaEnv {
  readonly VITE_BASE_URL: string;
  readonly VITE_IS_REQUEST_PROXY: 'true' | 'false';
  readonly VITE_API_URL: string;
  readonly VITE_API_URL_PREFIX: string;
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>;
  export default component;
}

declare module 'vue-router' {
  interface RouteMeta {
    title?: string;
    requiresAuth?: boolean;
    hideInMenu?: boolean;
    icon?: string;
    permission?: string;
    plugin?: string;
  }
}
