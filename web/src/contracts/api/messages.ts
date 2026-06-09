// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const MESSAGE_KEY = {
  AUTH_FORBIDDEN: 'auth.forbidden',
  COMMON_CONJUNCTION: 'common.conjunction',
  COMMON_COPYRIGHT: 'common.copyright',
} as const;

export type MessageKey = (typeof MESSAGE_KEY)[keyof typeof MESSAGE_KEY];
