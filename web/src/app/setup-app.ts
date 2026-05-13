import type { App as VueApp } from 'vue';

import { createPinia } from 'pinia';
import TDesign from 'tdesign-vue-next';

import { createAppRouter } from '@/router';

/**
 * Installs the minimal application runtime in explicit order:
 * Pinia first, then router, then TDesign.
 * This keeps route guards and stores predictable while the backend contracts are still static.
 */
export function setupApp(app: VueApp<Element>) {
  const pinia = createPinia();
  const router = createAppRouter(pinia);

  app.use(pinia);
  app.use(router);
  app.use(TDesign);
}
