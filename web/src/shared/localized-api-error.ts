// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { isHandledAuthRequestError } from '@/utils/auth-request-error';

export function localizedApiErrorMessage(
  translate: (key: string) => string,
  messageKey?: string,
  fallback?: string | null,
) {
  if (messageKey) {
    const translated = translate(messageKey);
    if (translated !== messageKey) {
      return translated;
    }
  }

  return fallback?.trim() || '';
}

function hasApiRequestErrorShape(error: unknown): error is {
  message: string;
  messageKey?: string;
  isApiRequestError: true;
} {
  return Boolean(
    error &&
    typeof error === 'object' &&
    'isApiRequestError' in error &&
    (error as { isApiRequestError?: unknown }).isApiRequestError === true &&
    'message' in error &&
    typeof (error as { message?: unknown }).message === 'string',
  );
}

export function resolveLocalizedErrorMessage(
  translate: (key: string) => string,
  error: unknown,
  fallbackMessage: string,
) {
  if (isHandledAuthRequestError(error)) {
    return '';
  }

  if (hasApiRequestErrorShape(error)) {
    return localizedApiErrorMessage(translate, error.messageKey, error.message) || fallbackMessage;
  }

  if (error instanceof Error) {
    const trimmed = error.message.trim();
    if (trimmed) {
      return trimmed;
    }
  }

  return fallbackMessage;
}
