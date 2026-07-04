import type { components as OpenAPIComponents } from '@/contracts/openapi/generated/schema';

export type PermissionListItem = OpenAPIComponents['schemas']['PermissionListItem'];
export type PermissionListResponse = OpenAPIComponents['schemas']['PermissionListResponse'];

export type PermissionFilters = {
  module?: string;
  keyword?: string;
};

export type PermissionDetailResponse = OpenAPIComponents['schemas']['PermissionDetailResponse'];
