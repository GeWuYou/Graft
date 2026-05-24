import { readErrorField } from '@/modules/shared/error-field';
import type { ApiRequestError } from '@/types/axios';

export type RoleFormField = 'name' | 'display' | 'description';
export type RolePermissionField = 'permission_ids';

const roleFormFieldMap: Record<string, RoleFormField> = {
  name: 'name',
  display: 'display',
  description: 'description',
};

const rolePermissionFieldMap: Record<string, RolePermissionField> = {
  permission_ids: 'permission_ids',
};

export function resolveRoleFormFieldError(error: ApiRequestError): RoleFormField | null {
  const field = readErrorField(error.responseData);
  if (!field) {
    return null;
  }

  return roleFormFieldMap[field] ?? null;
}

export function resolveRolePermissionFieldError(error: ApiRequestError): RolePermissionField | null {
  const field = readErrorField(error.responseData);
  if (!field) {
    return null;
  }

  return rolePermissionFieldMap[field] ?? null;
}
