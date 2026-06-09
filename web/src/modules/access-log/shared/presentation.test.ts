// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { accessLogPathSecondary, accessLogUserPrimary, accessLogUserSecondary } from './presentation';

const t = (key: string, params?: Record<string, unknown>) => {
  if (key === 'accessLog.user.userIdValue') {
    return `用户 ID：${params?.id}`;
  }

  return (
    {
      'accessLog.user.anonymous': '匿名用户',
      'accessLog.user.noUserId': '未关联用户 ID',
      'accessLog.user.unauthenticated': '未登录请求',
    }[key] ?? key
  );
};

describe('access-log presentation helpers', () => {
  it('hides duplicate route template', () => {
    expect(accessLogPathSecondary({ path: '/api/users', route: '/api/users' } as never)).toBe('');
  });

  it('shows route template when different from raw path', () => {
    expect(accessLogPathSecondary({ path: '/api/users/42', route: '/api/users/:id' } as never)).toBe('/api/users/:id');
  });

  it('formats anonymous user fallback', () => {
    expect(accessLogUserPrimary({ username: '' } as never, t as never)).toBe('匿名用户');
    expect(accessLogUserSecondary({ username: '', user_id: null } as never, t as never)).toBe('未登录请求');
  });

  it('formats identified user secondary text', () => {
    expect(accessLogUserPrimary({ username: 'alice' } as never, t as never)).toBe('alice');
    expect(accessLogUserSecondary({ username: 'alice', user_id: 12 } as never, t as never)).toBe('用户 ID：12');
  });
});
