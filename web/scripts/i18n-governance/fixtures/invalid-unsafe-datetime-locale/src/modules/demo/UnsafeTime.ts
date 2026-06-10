// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const label = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date());
export const fallback = new Date().toLocaleDateString();
export const voidLocale = Intl.DateTimeFormat(void 0, { timeStyle: 'short' }).format(new Date());
