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
  const mockEnabled = mode === 'mock' || mode === 'development';

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
    build: {
      chunkSizeWarningLimit: 1600,
      rollupOptions: {
        onwarn(warning, warn) {
          // `@vueuse/core` 当前版本产物会触发 Rollup 对 `#__PURE__` 注释位置的已知噪音，
          // 这里仅按精确来源收口日志，不吞掉其它依赖或业务代码 warning。
          if (
            warning.message.includes(
              'contains an annotation that Rollup cannot interpret due to the position of the comment',
            ) &&
            warning.id?.includes('/node_modules/@vueuse/core/dist/index.js')
          ) {
            return;
          }

          warn(warning);
        },
        output: {
          manualChunks(id) {
            if (!id.includes('node_modules')) {
              return undefined;
            }

            if (id.includes('/node_modules/tdesign-icons-vue-next/')) {
              return 'vendor-tdesign-icons';
            }

            if (id.includes('/node_modules/echarts/')) {
              return 'vendor-echarts';
            }

            if (id.includes('/node_modules/tdesign-vue-next/') || id.includes('/node_modules/tvision-color/')) {
              return 'vendor-tdesign';
            }

            if (
              id.includes('/node_modules/vue/') ||
              id.includes('/node_modules/@vue/') ||
              id.includes('/node_modules/vue-router/') ||
              id.includes('/node_modules/pinia/') ||
              id.includes('/node_modules/vue-i18n/') ||
              id.includes('/node_modules/@vueuse/core/')
            ) {
              return 'vendor-vue';
            }

            if (id.includes('/node_modules/lodash/')) {
              return 'vendor-utils';
            }

            return undefined;
          },
        },
      },
    },
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
      ...(mockEnabled
        ? [
            viteMockServe({
              mockPath: 'mock',
              enable: true,
            }),
          ]
        : []),
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
