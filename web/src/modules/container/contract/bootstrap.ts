// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { CONTAINER_ROUTE_PATH } from './paths';

export const CONTAINER_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: CONTAINER_ROUTE_PATH.LIST,
    routeName: 'ContainerList',
  },
} as const;

export type ContainerBootstrapRouteName =
  (typeof CONTAINER_BOOTSTRAP_ROUTE)[keyof typeof CONTAINER_BOOTSTRAP_ROUTE]['routeName'];
