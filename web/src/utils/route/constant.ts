// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const LAYOUT = () => import('@/layouts/index.vue');
export const BLANK_LAYOUT = () => import('@/layouts/blank.vue');

export const PAGE_NOT_FOUND_ROUTE = {
  path: '/:pathMatch(.*)*',
  name: '404Page',
  component: () => import('@/app/result/404/index.vue'),
  meta: {
    hidden: true,
  },
};
