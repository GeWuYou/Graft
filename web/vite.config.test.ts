// @vitest-environment node

import { afterEach, describe, expect, it } from 'vitest';

import { createViteConfig } from './vite.config';

function pluginNames(mode: string) {
  return (createViteConfig(mode).plugins ?? [])
    .map((plugin) => plugin?.name)
    .filter((name): name is string => Boolean(name));
}

function withRequestProxyEnabled(test: () => void) {
  const previousValue = process.env.VITE_IS_REQUEST_PROXY;
  process.env.VITE_IS_REQUEST_PROXY = 'true';

  try {
    test();
  } finally {
    if (previousValue === undefined) {
      delete process.env.VITE_IS_REQUEST_PROXY;
    } else {
      process.env.VITE_IS_REQUEST_PROXY = previousValue;
    }
  }
}

describe('createViteConfig', () => {
  afterEach(() => {
    delete process.env.VITE_ENABLE_MOCK;
  });

  it('does not enable mock in default development mode', () => {
    expect(pluginNames('development')).not.toContain('vite:mock');
  });

  it('replaces development preview routes with the empty release module', () => {
    const developmentConfig = createViteConfig('development');
    const releaseConfig = createViteConfig('release');
    const developmentAlias = developmentConfig.resolve?.alias?.['@/router/development-routes'];
    const releaseAlias = releaseConfig.resolve?.alias?.['@/router/development-routes'];

    expect(developmentAlias).toContain('development-routes.development.ts');
    expect(releaseAlias).toContain('development-routes.release.ts');
  });

  it('keeps explicit opt-in mock modes working', () => {
    expect(pluginNames('mock')).toContain('vite:mock');

    process.env.VITE_ENABLE_MOCK = 'true';
    expect(pluginNames('development')).toContain('vite:mock');
  });

  it('enables websocket proxying on the canonical api prefix when request proxy is enabled', () => {
    withRequestProxyEnabled(() => {
      const config = createViteConfig('development');
      const apiProxy = config.server?.proxy && '/api' in config.server.proxy ? config.server.proxy['/api'] : undefined;

      expect(typeof apiProxy).toBe('object');
      expect(apiProxy && 'ws' in apiProxy ? apiProxy.ws : undefined).toBe(true);
    });
  });

  it('proxies the core health endpoint without changing its root path', () => {
    withRequestProxyEnabled(() => {
      const config = createViteConfig('development');
      const healthProxy =
        config.server?.proxy && '/healthz' in config.server.proxy ? config.server.proxy['/healthz'] : undefined;

      expect(typeof healthProxy).toBe('object');
      expect(healthProxy && 'target' in healthProxy ? healthProxy.target : undefined).toEqual(expect.any(String));
      expect(healthProxy && 'changeOrigin' in healthProxy ? healthProxy.changeOrigin : undefined).toBe(true);
      expect(healthProxy && 'ws' in healthProxy ? healthProxy.ws : undefined).not.toBe(true);
    });
  });

  it('proxies MCP Explorer HTML and catalog routes without changing their root paths', () => {
    withRequestProxyEnabled(() => {
      const config = createViteConfig('development');

      for (const proxyPath of ['/mcp/docs', '/mcp/docs.json']) {
        const proxy =
          config.server?.proxy && proxyPath in config.server.proxy ? config.server.proxy[proxyPath] : undefined;

        expect(typeof proxy).toBe('object');
        expect(proxy && 'target' in proxy ? proxy.target : undefined).toEqual(expect.any(String));
        expect(proxy && 'changeOrigin' in proxy ? proxy.changeOrigin : undefined).toBe(true);
        expect(proxy && 'ws' in proxy ? proxy.ws : undefined).not.toBe(true);
      }
    });
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
