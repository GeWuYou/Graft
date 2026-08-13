import type { components as OpenAPIComponents } from '@/contracts/openapi/generated/schema';

export type PermissionListItem = OpenAPIComponents['schemas']['PermissionListItem'];
export type PermissionListResponse = OpenAPIComponents['schemas']['PermissionListResponse'];

export type PermissionFilters = {
  module?: string;
  keyword?: string;
  limit?: number;
  offset?: number;
};

export type PermissionDetailResponse = OpenAPIComponents['schemas']['PermissionDetailResponse'];
