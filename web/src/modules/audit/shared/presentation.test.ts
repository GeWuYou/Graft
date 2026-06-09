// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { resourceLabel } from './presentation';

function t(key: string) {
  if (key === 'audit.common.unknownResource') {
    return 'Unknown resource';
  }

  return key;
}

describe('audit presentation helpers', () => {
  it('uses request path before the empty secondary fallback for resource labels', () => {
    expect(resourceLabel({ request_path: '/api/auth/login' }, t)).toBe('/api/auth/login');
  });
});
