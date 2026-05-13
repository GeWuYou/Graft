import type { App as VueApp } from 'vue';

import { createPinia } from 'pinia';
import TDesign from 'tdesign-vue-next';

import { createAppRouter } from '@/router';

/**
 * 按显式顺序安装最小应用运行时：
 * 先注册 Pinia，再注册 router，最后挂载 TDesign，
 * 这样在后端契约仍是静态阶段时，路由守卫与共享状态的初始化顺序保持可预测。
 */
export function setupApp(app: VueApp<Element>) {
  const pinia = createPinia();
  const router = createAppRouter(pinia);

  app.use(pinia);
  app.use(router);
  app.use(TDesign);
}
