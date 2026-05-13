import { createPinia } from 'pinia';
import TDesign from 'tdesign-vue-next';
import type { App as VueApp } from 'vue';

import { setupApiClient } from '@/api/client';
import { setupI18n } from '@/app/i18n';
import { createAppRouter } from '@/router';

/**
 * 按显式顺序安装最小应用运行时：
 * 先注册 Pinia，再安装 i18n 和共享请求客户端，随后注册 router，最后挂载 TDesign，
 * 这样在后端契约仍是静态阶段时，路由守卫、locale 状态与请求头透传的初始化顺序保持可预测。
 */
export function setupApp(app: VueApp<Element>) {
  const pinia = createPinia();
  const router = createAppRouter(pinia);

  app.use(pinia);
  setupI18n(app, pinia);
  setupApiClient(pinia);
  app.use(router);
  app.use(TDesign);
}
