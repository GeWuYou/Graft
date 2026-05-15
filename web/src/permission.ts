import 'nprogress/nprogress.css'; // progress bar style

import NProgress from 'nprogress'; // progress bar
import { MessagePlugin } from 'tdesign-vue-next';
import type { RouteRecordRaw } from 'vue-router';

import router from '@/router';
import { getPermissionStore, useUserStore } from '@/store';
import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';

NProgress.configure({ showSpinner: false });

router.beforeEach(async (to, from, next) => {
  NProgress.start();

  const permissionStore = getPermissionStore();
  const { whiteListRouters } = permissionStore;

  const userStore = useUserStore();

  const initializeRoutes = async () => {
    const routeList = await permissionStore.buildAsyncRoutes();
    routeList.forEach((item: RouteRecordRaw) => {
      router.addRoute(item);
    });
  };

  if (userStore.token) {
    if (to.path === '/login') {
      next({ path: '/' });
      return;
    }
    try {
      const bootstrap = await userStore.ensureBootstrap();
      permissionStore.setBootstrapSnapshot(bootstrap);

      const { routesInitialized } = permissionStore;

      if (!routesInitialized) {
        await initializeRoutes();

        if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
          // 动态添加路由后，此处应当重定向到fullPath，否则会加载404页面内容
          next({ path: to.fullPath, replace: true, query: to.query });
        } else {
          const redirect = decodeURIComponent((from.query.redirect || to.path) as string);
          next(to.path === redirect ? { ...to, replace: true } : { path: redirect, query: to.query });
          return;
        }
      }
      if (to.name && router.hasRoute(to.name)) {
        next();
      } else {
        next(`/`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Login state expired';
      MessagePlugin.error(message);
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
      const bootstrap = await userStore.refreshToken().then(() => userStore.bootstrap(true));
      permissionStore.setBootstrapSnapshot(bootstrap);

      if (!permissionStore.routesInitialized) {
        await initializeRoutes();
      }

      if (to.path === '/login') {
        next({ path: '/' });
        return;
      }

      if (to.name === PAGE_NOT_FOUND_ROUTE.name) {
        next({ path: to.fullPath, replace: true, query: to.query });
      } else {
        next({ ...to, replace: true });
      }
      return;
    } catch {
      /* white list router */
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
