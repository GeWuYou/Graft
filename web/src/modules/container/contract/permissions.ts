// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const CONTAINER_PERMISSION_CODE = {
  VIEW: 'ops.container.view',
  DETAIL: 'ops.container.detail',
  LOGS: 'ops.container.logs',
  START: 'ops.container.start',
  STOP: 'ops.container.stop',
  RESTART: 'ops.container.restart',
} as const;

export type ContainerPermissionCode = (typeof CONTAINER_PERMISSION_CODE)[keyof typeof CONTAINER_PERMISSION_CODE];
