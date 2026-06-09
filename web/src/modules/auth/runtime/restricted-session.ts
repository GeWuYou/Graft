// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';

import { completeRequiredPasswordChange } from '@/modules/auth/api/auth';
import { AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import { resolveRuntimeHomePath } from '@/utils/route';

type CompleteRestrictedPasswordChangeOptions = {
  newPassword: string;
  bootstrap(force?: boolean): Promise<unknown>;
  buildAsyncRoutes(): Promise<RouteRecordRaw[]>;
  consumePendingRestrictedRedirect(fallbackPath: string): string;
  replace(path: string): Promise<unknown> | unknown;
};

// completeRestrictedPasswordChange 以 complete-required-password-change -> bootstrap -> rebuild routes 的顺序
// 恢复受限会话，避免前端本地伪造 must_change_password 状态或跳过最新 bootstrap 快照。
export async function completeRestrictedPasswordChange(options: CompleteRestrictedPasswordChangeOptions) {
  await completeRequiredPasswordChange({
    new_password: options.newPassword,
  });

  await options.bootstrap(true);
  const asyncRoutes = await options.buildAsyncRoutes();

  const fallbackPath = resolveRuntimeHomePath(asyncRoutes);
  const redirectPath = options.consumePendingRestrictedRedirect(fallbackPath);
  const normalizedRedirectPath = redirectPath
    ? (() => {
        try {
          const parsed = new URL(redirectPath, window.location.origin);
          if (parsed.origin !== window.location.origin) {
            return '';
          }
          return `${parsed.pathname}${parsed.search}${parsed.hash}`;
        } catch {
          return redirectPath.startsWith('/') ? redirectPath : '';
        }
      })()
    : '';
  const nextPath =
    !normalizedRedirectPath || normalizedRedirectPath === AUTH_ROUTE_PATH.RESTRICTED_SESSION
      ? fallbackPath
      : normalizedRedirectPath;

  await options.replace(nextPath);
}
