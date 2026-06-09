// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { API_CODE } from '@/contracts/api/codes';
import { readErrorField } from '@/modules/shared/error-field';
import type { ApiRequestError } from '@/types/axios';

export type UserFormField = 'username' | 'display' | 'password';
export type ResetPasswordField = 'password';

const createUserFieldMap: Record<string, UserFormField> = {
  username: 'username',
  display: 'display',
  password: 'password',
  new_password: 'password',
};

const updateUserFieldMap: Record<string, UserFormField> = {
  username: 'username',
  display: 'display',
};

const resetPasswordFieldMap: Record<string, ResetPasswordField> = {
  password: 'password',
  new_password: 'password',
};

export function resolveUserFormFieldError(error: ApiRequestError, mode: 'create' | 'edit'): UserFormField | null {
  const field = readErrorField(error.responseData);
  if (!field) {
    return mode === 'create' && error.code === API_CODE.AUTH_PASSWORD_POLICY_VIOLATION ? 'password' : null;
  }

  return (mode === 'create' ? createUserFieldMap : updateUserFieldMap)[field] ?? null;
}

export function resolveResetPasswordFieldError(error: ApiRequestError): ResetPasswordField | null {
  const field = readErrorField(error.responseData);
  if (!field) {
    return error.code === API_CODE.AUTH_PASSWORD_POLICY_VIOLATION ||
      error.code === API_CODE.AUTH_PASSWORD_REUSE_FORBIDDEN
      ? 'password'
      : null;
  }

  return resetPasswordFieldMap[field] ?? null;
}
