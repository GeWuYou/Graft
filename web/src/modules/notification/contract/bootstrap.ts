// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { NOTIFICATION_ROUTE_PATH } from './paths';

export const NOTIFICATION_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: NOTIFICATION_ROUTE_PATH.LIST,
    routeName: 'NotificationList',
  },
} as const;

export type NotificationBootstrapRouteName =
  (typeof NOTIFICATION_BOOTSTRAP_ROUTE)[keyof typeof NOTIFICATION_BOOTSTRAP_ROUTE]['routeName'];
