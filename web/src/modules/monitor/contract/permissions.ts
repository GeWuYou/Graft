// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const MONITOR_PERMISSION_CODE = {
  SERVER_STATUS_READ: 'monitor.server-status.read',
} as const;

export type MonitorPermissionCode = (typeof MONITOR_PERMISSION_CODE)[keyof typeof MONITOR_PERMISSION_CODE];
