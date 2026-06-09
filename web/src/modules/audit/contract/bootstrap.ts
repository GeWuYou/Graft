// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { AUDIT_ROUTE_PATH } from './paths';

export const AUDIT_BOOTSTRAP_ROUTE = {
  OVERVIEW: {
    menuPath: AUDIT_ROUTE_PATH.OVERVIEW,
    routeName: 'AuditOverview',
  },
  LOG_LIST: {
    menuPath: AUDIT_ROUTE_PATH.LOGS,
    routeName: 'AuditLogList',
  },
  INCIDENT_DETAIL: {
    menuPath: AUDIT_ROUTE_PATH.INCIDENT_DETAIL,
    routeName: 'AuditIncidentDetail',
  },
} as const;
