import type { RouteRecordRaw } from 'vue-router';

// release 构建将开发预览替换为空路由，确保 mock 页面和数据不会进入发行包。
export const developmentRouterList: RouteRecordRaw[] = [];
