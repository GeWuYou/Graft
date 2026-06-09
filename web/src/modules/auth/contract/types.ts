// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';

type AuthSchemas = components['schemas'];

export type LoginUser = AuthSchemas['LoginUser'];
export type LoginResponse = AuthSchemas['LoginResponse'];
export type BootstrapMenu = AuthSchemas['BootstrapMenu'];
export type BootstrapLocale = AuthSchemas['BootstrapLocale'];
export type BootstrapResponse = AuthSchemas['BootstrapResponse'];
export type LoginPayload = AuthSchemas['LoginRequest'];
export type ChangePasswordPayload = AuthSchemas['ChangePasswordRequest'];
export type CompleteRequiredPasswordChangePayload = AuthSchemas['CompleteRequiredPasswordChangeRequest'];
export type SessionSummary = AuthSchemas['SessionSummary'];
