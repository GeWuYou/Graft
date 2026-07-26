import 'nprogress/nprogress.css';

import NProgress from 'nprogress';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import type { Router, RouteRecordRaw } from 'vue-router';

import { t } from '@/locales';
import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import { useAuthSessionStore } from '@/modules/auth/store';
import router from '@/router';
import { finishRouteLoadingAfterRender, hideRouteLoading, startRouteLoading } from '@/router/route-loading';
import { emitDebugLog } from '@/shared/debug/runtime';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { getPermissionStore } from '@/store';
import { isRootEntryPath, resolveRuntimeHomePath, RUNTIME_ENTRY_FALLBACK_PATH } from '@/utils/route';
import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';

NProgress.configure({ showSpinner: false });

const DEVELOPMENT_PREVIEW_PATH = '/mock';
const DEVELOPMENT_PREVIEW_PATH_PREFIX = `${DEVELOPMENT_PREVIEW_PATH}/`;

/**
 * 判断目标路由是否与来源路由处于同一导航状态。
 *
 * 同状态导航不重复触发进度条，避免动态路由恢复或重复跳转时产生虚假的加载反馈。
 */
function isSameRouteStateNavigation(to: { name?: unknown; path: string }, from?: { name?: unknown; path?: string }) {
  if (!from?.path || to.path !== from.path) {
    return false;
  }

  if (to.name && from.name && to.name !== from.name) {
    return false;
  }

  return true;
}

/**
 * 收集动态路由及其一级子路由的名称。
 *
 * 这些名称用于在会话切换或 bootstrap 失败时按逆序卸载已挂载的动态路由。
 */
function collectBootstrapRouteNames(routes: RouteRecordRaw[]): string[] {
  const routeNames: string[] = [];

  for (const route of routes) {
    if (typeof route.name === 'string') {
      routeNames.push(route.name);
    }

    for (const child of route.children ?? []) {
      if (typeof child.name === 'string') {
        routeNames.push(child.name);
      }
    }
  }

  return routeNames;
}

function removeMountedBootstrapRoutes(targetRouter: Router, routes: RouteRecordRaw[]) {
  const routeNames = collectBootstrapRouteNames(routes).reverse();
  routeNames.forEach((routeName) => {
    if (targetRouter.hasRoute(routeName)) {
      targetRouter.removeRoute(routeName);
    }
  });
}

/**
 * 注册负责鉴权、会话恢复和动态路由初始化的路由守卫。
 *
 * 守卫必须先取得当前会话的 bootstrap 快照，再决定是否挂载菜单路由；恢复失败时会清理动态路由并回到登录页。
 *
 * @param targetRouter - 要注册守卫的 Vue Router 实例，默认使用应用根路由。
 */
export function registerRouteGuards(targetRouter: Router = router) {
  targetRouter.beforeEach(async (to, from, next) => {
    if (!isSameRouteStateNavigation(to, from)) {
      startRouteLoading();
      NProgress.start();
    }

    const permissionStore = getPermissionStore();
    const { whiteListRouters } = permissionStore;

    // 预览路由只在 Vite 开发构建注册，并在进入认证流程前短路，避免本地 UI 验收依赖后端会话。
    if (
      import.meta.env.DEV &&
      (to.path === DEVELOPMENT_PREVIEW_PATH || to.path.startsWith(DEVELOPMENT_PREVIEW_PATH_PREFIX))
    ) {
      next();
      return;
    }

    const userStore = useAuthSessionStore();
    emitDebugLog('navigation', 'guard-enter', {
      fromName: String(from.name ?? ''),
      fromPath: from.path,
      routesInitialized: permissionStore.routesInitialized,
      toName: String(to.name ?? ''),
      toPath: to.path,
      tokenPresent: Boolean(userStore.token),
    });

    // initializeRoutes 只在拿到最新 bootstrap 菜单快照后调用，确保动态路由
    // 与当前会话的后端菜单/权限结果保持一致，而不是复用旧的 demo 路由树。
    const initializeRoutes = async () => {
      removeMountedBootstrapRoutes(targetRouter, [...permissionStore.asyncRoutes, ...permissionStore.globalRoutes]);
      const routeList = await permissionStore.buildAsyncRoutes();
      routeList.forEach((item: RouteRecordRaw) => {
        targetRouter.addRoute(item);
      });
    };

    const isRestrictedSessionTarget =
      to.path === AUTH_ROUTE_PATH.RESTRICTED_SESSION || to.name === AUTH_ROUTE_NAME.RESTRICTED_SESSION;
    const isRestrictedSession = () => userStore.mustChangePassword;
    const redirectToRestrictedSession = () => {
      if (isRestrictedSessionTarget) {
        next();
        return;
      }

      userStore.setPendingRestrictedRedirect(to.fullPath);
      next({
        path: AUTH_ROUTE_PATH.RESTRICTED_SESSION,
        replace: true,
      });
    };

    if (userStore.token) {
      try {
        // 已有 access token 时优先保证 bootstrap 快照可用；这一步同时承担首次
        // 会话恢复职责，避免页面在缺少真实菜单/权限数据时继续导航。
        const bootstrap = await userStore.ensureBootstrap();
        permissionStore.setBootstrapSnapshot(bootstrap);

        const { routesInitialized } = permissionStore;

        if (!routesInitialized) {
          emitDebugLog('navigation', 'dynamic-routes-initialize', { targetPath: to.path });
          await initializeRoutes();
          emitDebugLog('navigation', 'dynamic-routes-initialized', {
            asyncRouteCount: permissionStore.asyncRoutes.length,
            globalRouteCount: permissionStore.globalRoutes.length,
          });

          if (isRestrictedSession()) {
            redirectToRestrictedSession();
            return;
          }

          if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
            // 动态路由挂载后必须重新匹配原始地址，否则首次导航仍会停留在占位 404 路由。
            next({ path: to.path, replace: true, query: to.query, hash: to.hash });
            return;
          } else {
            const redirect = decodeURIComponent((from.query.redirect || to.path) as string);
            emitDebugLog('navigation', 'redirect-after-route-initialization', {
              redirectPath: redirect,
              targetPath: to.path,
            });
            next(to.path === redirect ? { ...to, replace: true } : { path: redirect, query: to.query });
            return;
          }
        }

        if (to.path === AUTH_ROUTE_PATH.LOGIN || isRootEntryPath(to.path)) {
          if (isRestrictedSession()) {
            redirectToRestrictedSession();
            return;
          }

          if (to.path === AUTH_ROUTE_PATH.LOGIN) {
            emitDebugLog('navigation', 'redirect-authenticated-login', { targetPath: to.path });
            next({ path: resolveRuntimeHomePath(permissionStore.asyncRoutes), replace: true });
            return;
          }

          next();
          return;
        }

        if (isRestrictedSession()) {
          if (isRestrictedSessionTarget) {
            next();
            return;
          }

          redirectToRestrictedSession();
          return;
        }

        if (to.name && targetRouter.hasRoute(to.name)) {
          next();
        } else {
          emitDebugLog('navigation', 'redirect-route-not-mounted', { targetPath: to.path });
          next({ path: RUNTIME_ENTRY_FALLBACK_PATH, replace: true });
        }
      } catch (error) {
        const message = resolveLocalizedErrorMessage(t, error, t('app.auth.login.loginExpired'));
        MessagePlugin.error(message);
        // bootstrap 恢复失败意味着当前会话无法再信任，需要同时清理本地 token
        // 和已挂载的动态路由，再把用户送回登录页重新建立会话。
        removeMountedBootstrapRoutes(targetRouter, [...permissionStore.asyncRoutes, ...permissionStore.globalRoutes]);
        userStore.clearSessionState();
        permissionStore.restoreRoutes();
        emitDebugLog('navigation', 'redirect-bootstrap-failed', { targetPath: to.path });
        next({
          path: AUTH_ROUTE_PATH.LOGIN,
          query: { redirect: encodeURIComponent(to.fullPath) },
        });
        NProgress.done();
      }
    } else {
      try {
        // 本地没有 access token 时，仍允许先用 refresh cookie 静默恢复一次会话；
        // 只有 refresh 失败后才退回白名单/登录页，避免强制打断仍然有效的登录态。
        await userStore.refreshToken();
        const bootstrap = await userStore.bootstrap(true);
        permissionStore.setBootstrapSnapshot(bootstrap);

        if (!permissionStore.routesInitialized) {
          emitDebugLog('navigation', 'dynamic-routes-initialize-after-refresh', { targetPath: to.path });
          await initializeRoutes();
        }

        if (isRestrictedSession()) {
          redirectToRestrictedSession();
          return;
        }

        if (to.path === AUTH_ROUTE_PATH.LOGIN || isRootEntryPath(to.path)) {
          if (to.path === AUTH_ROUTE_PATH.LOGIN) {
            emitDebugLog('navigation', 'redirect-refreshed-login', { targetPath: to.path });
            next({ path: resolveRuntimeHomePath(permissionStore.asyncRoutes), replace: true });
            return;
          }

          next();
          return;
        }

        if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
          emitDebugLog('navigation', 'retry-not-found-after-refresh', { targetPath: to.path });
          next({ path: to.path, replace: true, query: to.query, hash: to.hash });
        } else {
          emitDebugLog('navigation', 'retry-target-after-refresh', { targetPath: to.path });
          next({ ...to, replace: true });
        }
        return;
      } catch {
        // 无法静默恢复时，仅保留白名单路径直达，其它路径统一回登录页重建会话。
        if (whiteListRouters.includes(to.path)) {
          emitDebugLog('navigation', 'allow-whitelist-without-session', { targetPath: to.path });
          next();
        } else {
          emitDebugLog('navigation', 'redirect-login-without-session', { targetPath: to.path });
          next({
            path: AUTH_ROUTE_PATH.LOGIN,
            query: { redirect: encodeURIComponent(to.fullPath) },
          });
        }
      }
      NProgress.done();
    }
  });

  targetRouter.afterEach((to, from) => {
    emitDebugLog('navigation', 'guard-complete', {
      fromName: String(from?.name ?? ''),
      fromPath: from?.path ?? '',
      toName: String(to?.name ?? ''),
      toPath: to?.path ?? '',
    });
    if (to.path === AUTH_ROUTE_PATH.LOGIN) {
      const userStore = useAuthSessionStore();
      const permissionStore = getPermissionStore();

      removeMountedBootstrapRoutes(targetRouter, [...permissionStore.asyncRoutes, ...permissionStore.globalRoutes]);
      userStore.clearSessionState();
      permissionStore.restoreRoutes();
    }
    NProgress.done();
    if (!isSameRouteStateNavigation(to, from)) {
      void finishRouteLoadingAfterRender();
    }
  });

  targetRouter.onError(() => {
    NProgress.done();
    hideRouteLoading();
  });
}
