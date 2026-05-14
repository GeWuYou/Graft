import fs from 'node:fs';
import path from 'node:path';

import UnoCSS from '@unocss/vite';
import vue from '@vitejs/plugin-vue';
import vueJsx from '@vitejs/plugin-vue-jsx';
import { type ConfigEnv, defineConfig, loadEnv, type ProxyOptions, type UserConfig } from 'vite';
import { viteMockServe } from 'vite-plugin-mock';
import svgLoader from 'vite-svg-loader';

const CWD = process.cwd();
const lessVariablesFile = path.resolve(CWD, 'src/style/variables.less');

export function createViteConfig(mode: string): UserConfig {
  const env = loadEnv(mode, CWD, '');
  const base = env.VITE_BASE_URL || '/';
  const apiPrefix = env.VITE_API_URL_PREFIX || '/api';
  const apiTarget = env.VITE_API_URL || 'http://127.0.0.1:3000';
  const proxyEnabled = env.VITE_IS_REQUEST_PROXY === 'true';

  const lessOptions = {
    javascriptEnabled: true,
    math: 'strict' as const,
    ...(fs.existsSync(lessVariablesFile)
      ? {
          modifyVars: {
            // 当前 `web/src` 还未完全切到 starter 的 less 体系时，避免强制引用不存在的变量文件。
            hack: `true; @import (reference) "${lessVariablesFile.replaceAll('\\', '/')}";`,
          },
        }
      : {}),
  };

  return {
    base,
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    css: {
      preprocessorOptions: {
        less: lessOptions,
      },
    },
    plugins: [
      vue(),
      vueJsx(),
      UnoCSS(),
      svgLoader(),
      viteMockServe({
        mockPath: 'mock',
        enable: true,
      }),
    ],
    server: {
      port: 3002,
      host: '0.0.0.0',
      allowedHosts: true,
      proxy: proxyEnabled
        ? ({
            [apiPrefix]: {
              target: apiTarget,
              changeOrigin: true,
            } satisfies ProxyOptions,
          } as Record<string, string | ProxyOptions>)
        : undefined,
    },
    preview: {
      host: '0.0.0.0',
      port: 4173,
    },
  };
}

export default defineConfig(({ mode }: ConfigEnv) => createViteConfig(mode));
