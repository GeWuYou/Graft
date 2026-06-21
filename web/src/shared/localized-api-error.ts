// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { isHandledAuthRequestError } from '@/utils/auth-request-error';

/**
 * 获取本地化的 API 错误消息。
 *
 * @param translate - 将消息键转换为本地化字符串的函数
 * @returns 当 `messageKey` 被成功翻译时返回翻译后的字符串；否则返回修剪后的 `fallback`，若其不存在或为空，则返回空字符串
 */
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

/**
 * 从错误对象中解析本地化的错误消息。
 *
 * @returns 解析得到的错误消息字符串。若为已处理的认证错误返回空字符串；若为 API 请求错误优先返回本地化消息，否则返回备用文本；若为标准 Error 对象返回其消息；其他情况返回备用文本。
 */
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
