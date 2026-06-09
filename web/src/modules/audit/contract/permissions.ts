// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const AUDIT_PERMISSION_CODE = {
  READ: 'audit.read',
} as const;

export type AuditPermissionCode = (typeof AUDIT_PERMISSION_CODE)[keyof typeof AUDIT_PERMISSION_CODE];
