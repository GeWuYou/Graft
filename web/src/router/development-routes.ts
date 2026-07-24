// TypeScript 的路径解析不读取 Vite alias；默认解析 release 空路由，实际构建由 Vite 按模式替换。
export { developmentRouterList } from './development-routes.release';
