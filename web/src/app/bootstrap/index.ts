/* eslint-disable simple-import-sort/imports */
import { createApp } from 'vue';
import TDesign from 'tdesign-vue-next';

import App from '@/App.vue';
import { i18n } from '@/locales';
import router from '@/router';
import { store, useTabsRouterStore } from '@/store';
import { createLogger, patchGlobalLoggerContext } from '@/utils/logger';

import { registerPermissionDirective } from './permission-directive';
import { registerRouteGuards } from './route-guards';

import 'tdesign-vue-next/es/style/index.css';
import '@/style/index.less';

const appLogger = createLogger('app.runtime').withContext({
  component: 'app.bootstrap',
});

// bootstrapApp owns the single startup path for the real web runtime.
export function bootstrapApp() {
  registerRouteGuards(router);
  syncRouteLoggerContext(router.currentRoute.value.path);
  registerGlobalLoggerSinks();
  router.afterEach((to) => {
    syncRouteLoggerContext(to.fullPath || to.path);
  });

  const app = createApp(App);
  const tabsRouterStore = useTabsRouterStore(store);

  // 自愈上一次异常中断时遗留的 tabs 刷新态，避免内容区被全局 loading 永久覆盖。
  tabsRouterStore.healPersistedState();

  app.use(TDesign);
  app.use(store);
  app.use(router);
  app.use(i18n);
  // 权限指令只消费 bootstrap 权限快照，不引入第二套前端鉴权真值。
  registerPermissionDirective(app);
  app.config.errorHandler = (error, instance, info) => {
    appLogger.error(normalizeError(error), {
      component: resolveComponentName(instance),
      eventType: 'vue.error',
      info,
    });
  };

  app.mount('#app');

  return app;
}

function registerGlobalLoggerSinks() {
  if (typeof window === 'undefined') {
    return;
  }

  window.addEventListener('error', (event) => {
    appLogger.error(normalizeError(event.error ?? event.message), {
      component: 'window',
      eventType: 'window.error',
      filename: event.filename,
      line: event.lineno,
      column: event.colno,
    });
  });

  window.addEventListener('unhandledrejection', (event) => {
    appLogger.error(normalizeError(event.reason), {
      component: 'window',
      eventType: 'window.unhandledrejection',
    });
  });
}

function syncRouteLoggerContext(route: string) {
  patchGlobalLoggerContext({
    route: route.trim(),
  });
}

function resolveComponentName(instance: unknown) {
  if (!instance || typeof instance !== 'object') {
    return 'vue.app';
  }

  const candidate = instance as {
    type?: {
      name?: string;
      __name?: string;
    };
  };

  return candidate.type?.name || candidate.type?.__name || 'vue.app';
}

function normalizeError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }

  if (typeof error === 'string' && error.trim()) {
    return new Error(error.trim());
  }

  return new Error('Unexpected runtime error');
}
