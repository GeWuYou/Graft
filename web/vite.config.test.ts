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

  it('enables websocket proxying on the canonical api prefix when request proxy is enabled', () => {
    process.env.VITE_IS_REQUEST_PROXY = 'true';
    try {
      const config = createViteConfig('development');
      const apiProxy = config.server?.proxy && '/api' in config.server.proxy ? config.server.proxy['/api'] : undefined;

      expect(typeof apiProxy).toBe('object');
      expect(apiProxy && 'ws' in apiProxy ? apiProxy.ws : undefined).toBe(true);
    } finally {
      delete process.env.VITE_IS_REQUEST_PROXY;
    }
  });

  it('proxies the core health endpoint without changing its root path', () => {
    process.env.VITE_IS_REQUEST_PROXY = 'true';
    try {
      const config = createViteConfig('development');
      const healthProxy =
        config.server?.proxy && '/healthz' in config.server.proxy ? config.server.proxy['/healthz'] : undefined;

      expect(typeof healthProxy).toBe('object');
      expect(healthProxy && 'target' in healthProxy ? healthProxy.target : undefined).toEqual(expect.any(String));
      expect(healthProxy && 'changeOrigin' in healthProxy ? healthProxy.changeOrigin : undefined).toBe(true);
      expect(healthProxy && 'ws' in healthProxy ? healthProxy.ws : undefined).not.toBe(true);
    } finally {
      delete process.env.VITE_IS_REQUEST_PROXY;
    }
  });

  it('splits monaco dependencies into a dedicated vendor chunk', () => {
    const config = createViteConfig('development');
    const manualChunks = config.build?.rollupOptions?.output;

    expect(manualChunks).toBeDefined();
    if (!manualChunks || Array.isArray(manualChunks) || typeof manualChunks.manualChunks !== 'function') {
      throw new Error('expected manualChunks function to be configured');
    }

    expect(manualChunks.manualChunks('/tmp/node_modules/monaco-editor/esm/vs/editor/editor.api.js')).toBe(
      'vendor-monaco',
    );
    expect(manualChunks.manualChunks('/tmp/node_modules/monaco-yaml/lib/esm/index.js')).toBe('vendor-monaco');
  });
});
