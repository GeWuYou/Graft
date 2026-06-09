// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { APP_LOG_ROUTE_PATH } from './paths';

export const APP_LOG_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: APP_LOG_ROUTE_PATH.LIST,
    routeName: 'AppLogList',
  },
} as const;
