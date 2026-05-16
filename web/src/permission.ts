import 'nprogress/nprogress.css'; // progress bar style

import NProgress from 'nprogress'; // progress bar
import { MessagePlugin } from 'tdesign-vue-next';
import type { RouteRecordRaw } from 'vue-router';

import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/contracts/auth/routes';
import router from '@/router';
import { getPermissionStore, useUserStore } from '@/store';
import { isRootEntryPath, resolveRuntimeHomePath, RUNTIME_ENTRY_FALLBACK_PATH } from '@/utils/route';
import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';

NProgress.configure({ showSpinner: false });

router.beforeEach(async (to, from, next) => {
  NProgress.start();

  const permissionStore = getPermissionStore();
  const { whiteListRouters } = permissionStore;

  const userStore = useUserStore();

  // initializeRoutes 只在拿到最新 bootstrap 菜单快照后调用，确保动态路由
  // 与当前会话的后端菜单/权限结果保持一致，而不是复用旧的 demo 路由树。
  const initializeRoutes = async () => {
    const routeList = await permissionStore.buildAsyncRoutes();
    routeList.forEach((item: RouteRecordRaw) => {
      router.addRoute(item);
    });
  };

  const isRestrictedSessionTarget =
    to.path === AUTH_ROUTE_PATH.RESTRICTED_SESSION || to.name === AUTH_ROUTE_NAME.RESTRICTED_SESSION;
  const isRestrictedSession = () => userStore.mustChangePassword;
  const redirectToRestrictedSession = () => {
    if (!isRestrictedSessionTarget && to.fullPath !== AUTH_ROUTE_PATH.RESTRICTED_SESSION) {
      userStore.setPendingRestrictedRedirect(to.fullPath);
    }

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
        await initializeRoutes();

        if (isRestrictedSession()) {
          redirectToRestrictedSession();
          return;
        }

        if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
          // 动态添加路由后，此处应当重定向到fullPath，否则会加载404页面内容
          next({ path: to.fullPath, replace: true, query: to.query });
          return;
        } else {
          const redirect = decodeURIComponent((from.query.redirect || to.path) as string);
          next(to.path === redirect ? { ...to, replace: true } : { path: redirect, query: to.query });
          return;
        }
      }

      const runtimeHomePath = resolveRuntimeHomePath(permissionStore.asyncRoutes);
      if (to.path === '/login' || isRootEntryPath(to.path)) {
        if (isRestrictedSession()) {
          redirectToRestrictedSession();
          return;
        }

        next({ path: runtimeHomePath, replace: true });
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

      if (to.name && router.hasRoute(to.name)) {
        next();
      } else {
        next({ path: RUNTIME_ENTRY_FALLBACK_PATH, replace: true });
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Login state expired';
      MessagePlugin.error(message);
      // bootstrap 恢复失败意味着当前会话无法再信任，需要同时清理本地 token
      // 和已挂载的动态路由，再把用户送回登录页重新建立会话。
      userStore.clearSessionState();
      permissionStore.restoreRoutes();
      next({
        path: '/login',
        query: { redirect: encodeURIComponent(to.fullPath) },
      });
      NProgress.done();
    }
  } else {
    try {
      // 本地没有 access token 时，仍允许先用 refresh cookie 静默恢复一次会话；
      // 只有 refresh 失败后才退回白名单/登录页，避免强制打断仍然有效的登录态。
      const bootstrap = await userStore.refreshToken().then(() => userStore.bootstrap(true));
      permissionStore.setBootstrapSnapshot(bootstrap);

      if (!permissionStore.routesInitialized) {
        await initializeRoutes();
      }

      if (isRestrictedSession()) {
        redirectToRestrictedSession();
        return;
      }

      const runtimeHomePath = resolveRuntimeHomePath(permissionStore.asyncRoutes);
      if (to.path === '/login' || isRootEntryPath(to.path)) {
        next({ path: runtimeHomePath, replace: true });
        return;
      }

      if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
        next({ path: to.fullPath, replace: true, query: to.query });
      } else {
        next({ ...to, replace: true });
      }
      return;
    } catch {
      // 无法静默恢复时，仅保留白名单路径直达，其它路径统一回登录页重建会话。
      if (whiteListRouters.includes(to.path)) {
        next();
      } else {
        next({
          path: '/login',
          query: { redirect: encodeURIComponent(to.fullPath) },
        });
      }
    }
    NProgress.done();
  }
});

router.afterEach((to) => {
  if (to.path === '/login') {
    const userStore = useUserStore();
    const permissionStore = getPermissionStore();

    userStore.clearSessionState();
    permissionStore.restoreRoutes();
  }
  NProgress.done();
});
