/* eslint-disable simple-import-sort/imports */
import { createApp } from 'vue';

import App from '@/App.vue';
import { i18n } from '@/locales';
import { isProjectMonacoBenignCancellationError } from '@/modules/project/shared/project-monaco-debug';
import { initDebugRuntime } from '@/shared/debug/runtime';
import router from '@/router';
import { store, useTabsRouterStore } from '@/store';
import { isHandledAuthRequestError } from '@/utils/auth-request-error';
import { createLogger, patchGlobalLoggerContext } from '@/utils/logger';

import { registerPermissionDirective } from './permission-directive';
import { registerRouteGuards } from './route-guards';

import '@/style/index.less';

const appLogger = createLogger('app.runtime').withContext({
  component: 'app.bootstrap',
});

const RESIZE_OBSERVER_LOOP_ERROR_MESSAGES = new Set([
  'ResizeObserver loop completed with undelivered notifications.',
  'ResizeObserver loop limit exceeded',
]);

/**
 * 初始化并挂载应用。
 *
 * @returns 创建并挂载后的 Vue 应用实例
 */
export function bootstrapApp() {
  registerRouteGuards(router);
  syncRouteLoggerContext(router.currentRoute.value.path);
  registerGlobalLoggerSinks();
  router.afterEach((to) => {
    syncRouteLoggerContext(to.fullPath || to.path);
  });

  const app = createApp(App);
  app.use(store);
  initDebugRuntime();
  const tabsRouterStore = useTabsRouterStore(store);

  // 必须在 app.use(store) 之后再创建带 persist 的 store，避免启动阶段拿到未 hydrate 的初始 tabs 状态。
  tabsRouterStore.healPersistedState();

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

/**
 * 在浏览器环境中注册全局运行时错误监听器。
 *
 * 记录未被抑制的脚本错误和未处理的 Promise 拒绝，并拦截 ResizeObserver 循环错误、已处理的鉴权请求错误及 Monaco 良性取消错误。
 */
function registerGlobalLoggerSinks() {
  if (typeof window === 'undefined') {
    return;
  }

  window.addEventListener('error', (event) => {
    const error = event.error ?? event.message;
    if (isResizeObserverLoopError(error)) {
      event.preventDefault();
      return;
    }

    appLogger.error(normalizeError(error), {
      component: 'window',
      eventType: 'window.error',
      filename: event.filename,
      line: event.lineno,
      column: event.colno,
    });
  });

  window.addEventListener('unhandledrejection', (event) => {
    if (isHandledAuthRequestError(event.reason)) {
      event.preventDefault();
      return;
    }

    if (isProjectMonacoBenignCancellationError(event.reason)) {
      event.preventDefault();
      return;
    }

    appLogger.error(normalizeError(event.reason), {
      component: 'window',
      eventType: 'window.unhandledrejection',
    });
  });
}

/**
 * 判断错误是否属于 ResizeObserver 循环类错误。
 *
 * @param error - 待检查的错误值
 * @returns `true` if 错误消息匹配已知的 ResizeObserver 循环错误，`false` otherwise
 */
function isResizeObserverLoopError(error: unknown) {
  const message = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
  return RESIZE_OBSERVER_LOOP_ERROR_MESSAGES.has(message.trim());
}

/**
 * 将当前路由同步到全局日志上下文。
 *
 * @param route - 当前路由路径
 */
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
