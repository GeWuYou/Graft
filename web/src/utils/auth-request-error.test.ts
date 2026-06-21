import { describe, expect, it } from 'vitest';

import { API_CODE } from '@/contracts/api/codes';

import { isHandledAuthRequestError } from './auth-request-error';

describe('isHandledAuthRequestError', () => {
  it('returns true for handled 401 auth request errors', () => {
    expect(
      isHandledAuthRequestError({
        isApiRequestError: true,
        status: 401,
        code: API_CODE.AUTH_TOKEN_EXPIRED,
      }),
    ).toBe(true);
    expect(
      isHandledAuthRequestError({
        isApiRequestError: true,
        status: 401,
        code: API_CODE.AUTH_TOKEN_INVALID,
      }),
    ).toBe(true);
    expect(
      isHandledAuthRequestError({
        isApiRequestError: true,
        status: 401,
        code: API_CODE.AUTH_TOKEN_MISSING,
      }),
    ).toBe(true);
  });

  it('returns false for unrelated errors', () => {
    expect(
      isHandledAuthRequestError({
        isApiRequestError: true,
        status: 403,
        code: API_CODE.AUTH_FORBIDDEN,
      }),
    ).toBe(false);
    expect(isHandledAuthRequestError(new Error('boom'))).toBe(false);
    expect(isHandledAuthRequestError(null)).toBe(false);
  });
});
