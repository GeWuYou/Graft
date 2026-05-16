import type { RouteRecordRaw } from 'vue-router';

import { completeRequiredPasswordChange } from '@/api/auth';
import { AUTH_ROUTE_PATH } from '@/contracts/auth/routes';
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
  const nextPath = redirectPath === AUTH_ROUTE_PATH.RESTRICTED_SESSION ? fallbackPath : redirectPath;

  await options.replace(nextPath);
}
