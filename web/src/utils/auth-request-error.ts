// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { API_CODE } from '@/contracts/api/codes';

type MinimalApiRequestError = {
  code?: string;
  isApiRequestError?: boolean;
  status?: number;
};

export function isHandledAuthRequestError(error: unknown) {
  if (!error || typeof error !== 'object') {
    return false;
  }

  const candidate = error as MinimalApiRequestError;
  if (candidate.isApiRequestError !== true || candidate.status !== 401) {
    return false;
  }

  return (
    candidate.code === API_CODE.AUTH_TOKEN_EXPIRED ||
    candidate.code === API_CODE.AUTH_TOKEN_INVALID ||
    candidate.code === API_CODE.AUTH_TOKEN_MISSING
  );
}
