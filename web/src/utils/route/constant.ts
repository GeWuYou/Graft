export const LAYOUT = () => import('@/layouts/index.vue');

export const PAGE_NOT_FOUND_ROUTE = {
  path: '/:pathMatch(.*)*',
  name: '404Page',
  component: () => import('@/app/result/404/index.vue'),
  meta: {
    hidden: true,
  },
};
