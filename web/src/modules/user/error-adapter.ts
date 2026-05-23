import { API_CODE } from '@/contracts/api/codes';
import type { ApiRequestError } from '@/types/axios';

export type UserFormField = 'username' | 'display' | 'password';

const createUserFieldMap: Record<string, UserFormField> = {
  username: 'username',
  display: 'display',
  password: 'password',
  new_password: 'password',
};

export function resolveCreateUserFieldError(error: ApiRequestError): UserFormField | null {
  const field = readField(error.responseData);
  if (!field) {
    return error.code === API_CODE.AUTH_PASSWORD_POLICY_VIOLATION ? 'password' : null;
  }

  return createUserFieldMap[field] ?? null;
}

function readField(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object' || !('data' in payload)) {
    return null;
  }

  const data = (payload as { data?: unknown }).data;
  if (!data || typeof data !== 'object' || !('field' in data)) {
    return null;
  }

  const field = (data as { field?: unknown }).field;
  return typeof field === 'string' && field.trim() !== '' ? field : null;
}
