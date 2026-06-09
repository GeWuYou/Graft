// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { SCHEDULED_TASK_ROUTE_PATH } from './paths';

export const SCHEDULED_TASK_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: SCHEDULED_TASK_ROUTE_PATH.LIST,
    routeName: 'ScheduledTaskList',
  },
} as const;

export type ScheduledTaskBootstrapRouteName =
  (typeof SCHEDULED_TASK_BOOTSTRAP_ROUTE)[keyof typeof SCHEDULED_TASK_BOOTSTRAP_ROUTE]['routeName'];
