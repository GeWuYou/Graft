// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components as OpenAPIComponents } from '@/contracts/openapi/generated/schema';

export type PermissionListItem = OpenAPIComponents['schemas']['PermissionListItem'];
export type PermissionListResponse = OpenAPIComponents['schemas']['PermissionListResponse'];

export type PermissionFilters = {
  category?: string;
  keyword?: string;
};

export type PermissionDetailResponse = OpenAPIComponents['schemas']['PermissionDetailResponse'];
