// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';

export type NotificationSeverity = components['schemas']['notification-severity'];

const NOTIFICATION_SEVERITY = {
  INFO: 'info',
  WARNING: 'warning',
  ERROR: 'error',
  CRITICAL: 'critical',
} as const satisfies Record<string, NotificationSeverity>;

export const NOTIFICATION_SEVERITY_VALUES = Object.values(NOTIFICATION_SEVERITY);
