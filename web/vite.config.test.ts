// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// @vitest-environment node

import { afterEach, describe, expect, it } from 'vitest';

import { createViteConfig } from './vite.config';

function pluginNames(mode: string) {
  return (createViteConfig(mode).plugins ?? [])
    .map((plugin) => plugin?.name)
    .filter((name): name is string => Boolean(name));
}

describe('createViteConfig', () => {
  afterEach(() => {
    delete process.env.VITE_ENABLE_MOCK;
  });

  it('does not enable mock in default development mode', () => {
    expect(pluginNames('development')).not.toContain('vite:mock');
  });

  it('keeps explicit opt-in mock modes working', () => {
    expect(pluginNames('mock')).toContain('vite:mock');

    process.env.VITE_ENABLE_MOCK = 'true';
    expect(pluginNames('development')).toContain('vite:mock');
  });
});
