// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { maskSensitiveJson } from './json-sanitize';

describe('json-sanitize', () => {
  it('masks sensitive fields in nested objects and arrays', () => {
    expect(
      maskSensitiveJson({
        user: 'admin',
        password: 'p1',
        nested: {
          token: 't1',
          safe: 'visible',
        },
        items: [{ api_key: 'key-1' }, { authorization: 'Bearer token' }],
      }),
    ).toEqual({
      user: 'admin',
      password: '******',
      nested: {
        token: '******',
        safe: 'visible',
      },
      items: [{ api_key: '******' }, { authorization: '******' }],
    });
  });
});
