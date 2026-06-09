// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { LoggerTransport } from '@/utils/logger/types';

export const noopTransport: LoggerTransport = {
  log() {},
};
