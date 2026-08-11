/* eslint-disable @typescript-eslint/no-unused-vars */
/// <reference types="vite/client" />

export {};

interface ImportMetaEnv {
  readonly VITE_BASE_URL: string;
  readonly VITE_IS_REQUEST_PROXY: 'true' | 'false';
  readonly VITE_API_TARGET: string;
  readonly VITE_API_URL_PREFIX: string;
  readonly VITE_LOG_LEVEL?: string;
  readonly VITE_DEBUG_NAVIGATION?: 'true' | 'false';
  readonly VITE_DEBUG_TABS?: 'true' | 'false';
  readonly VITE_DEBUG_TABS_LAYOUT?: 'true' | 'false';
  readonly VITE_DEBUG_TABS_STORE?: 'true' | 'false';
  readonly VITE_DEBUG_MANAGEMENT_TABLE_LAYOUT?: 'true' | 'false';
  readonly VITE_DEBUG_PROJECT_LOGS?: 'true' | 'false';
  readonly VITE_DEBUG_PROJECT_MONACO?: 'true' | 'false';
  readonly VITE_DEBUG_PROJECT_TEMPLATES?: 'true' | 'false';
  readonly VITE_DEBUG_PROJECT_WORKSPACE?: 'true' | 'false';
  readonly VITE_DEBUG_CONTAINER_RAW_JSON?: 'true' | 'false';
}

declare global {
  interface Window {
    __GRAFT_DEBUG__?: {
      clear: (flagId?: string) => Record<string, boolean>;
      disable: (flagId: string) => boolean;
      enable: (flagId: string) => boolean;
      help: () => string;
      isEnabled: (flagId: string) => boolean;
      list: () => Array<{
        defaultEnabled: boolean;
        effectiveEnabled: boolean;
        envKeys?: readonly string[];
        flagId: string;
        owner: string;
        parentFlagId?: string;
        relatedPaths: string[];
        summary: string;
        runtimeOverride: boolean | null;
      }>;
      set: (flagId: string, value: boolean) => boolean;
      state: () => {
        dangerousCapabilitiesAvailable: boolean;
        enabled: boolean;
        flags: Record<string, boolean>;
        runtimeOverrides: Record<string, boolean>;
      };
    };
  }
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
