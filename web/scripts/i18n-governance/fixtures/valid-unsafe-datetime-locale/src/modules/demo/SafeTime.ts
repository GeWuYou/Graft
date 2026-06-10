// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { formatLocaleDateTime } from '@/shared/observability';

export const label = formatLocaleDateTime('2026-06-10T02:38:00Z', 'en-US');
export const fallback = new Date().toLocaleDateString('en-US');
export const count = 1234;
export const numberLabel = count.toLocaleString();
