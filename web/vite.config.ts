import fs from 'node:fs';
import path from 'node:path';

import { TDesignResolver } from '@tdesign-vue-next/auto-import-resolver';
import UnoCSS from '@unocss/vite';
import vue from '@vitejs/plugin-vue';
import vueJsx from '@vitejs/plugin-vue-jsx';
import AutoImport from 'unplugin-auto-import/vite';
import type { ComponentInfo, ComponentResolveResult, ComponentResolverObject } from 'unplugin-vue-components';
import Components from 'unplugin-vue-components/vite';
import { type ConfigEnv, defineConfig, loadEnv, type ProxyOptions, type UserConfig } from 'vite';
import { viteMockServe } from 'vite-plugin-mock';
import svgLoader from 'vite-svg-loader';

const CWD = process.cwd();
const lessVariablesFile = path.resolve(CWD, 'src/style/variables.less');
const monacoEditorMainCssFile = path.resolve(CWD, 'node_modules/monaco-editor/min/vs/editor/editor.main.css');
const monacoEditorEsmVsDirectory = path.resolve(CWD, 'node_modules/monaco-editor/esm/vs');
const tdesignVueNextPackageName = 'tdesign-vue-next';
const tdesignVueNextEsmEntry = `${tdesignVueNextPackageName}/esm`;
const tdesignComponentPathOverrides: Record<string, string> = {
  Aside: 'layout',
  BreadcrumbItem: 'breadcrumb',
  CheckTag: 'tag',
  CheckTagGroup: 'tag',
  CheckboxGroup: 'checkbox',
  Col: 'grid',
  CollapsePanel: 'collapse',
  Content: 'layout',
  DateRangePicker: 'date-picker',
  DescriptionsItem: 'descriptions',
  DropdownItem: 'dropdown',
  DropdownMenu: 'dropdown',
  Footer: 'layout',
  FormItem: 'form',
  Header: 'layout',
  HeadMenu: 'menu',
  ListItem: 'list',
  MenuItem: 'menu',
  Option: 'select',
  OptionGroup: 'select',
  QRCode: 'qrcode',
  RadioButton: 'radio',
  RadioGroup: 'radio',
  Row: 'grid',
  Submenu: 'menu',
  TabPanel: 'tabs',
  TimelineItem: 'timeline',
};

function isComponentInfo(result: ComponentResolveResult): result is ComponentInfo {
  return Boolean(result && typeof result === 'object' && !('then' in result) && 'from' in result);
}

function createTDesignVueNextResolver(): ComponentResolverObject {
  const resolver = TDesignResolver({
    library: 'vue-next',
    esm: true,
  }).find((item) => typeof item === 'object' && item.type === 'component');

  if (!resolver || typeof resolver === 'function') {
    throw new Error('TDesign component resolver is unavailable');
  }

  return {
    ...resolver,
    resolve(name) {
      const resolved = resolver.resolve?.(name);

      if (!isComponentInfo(resolved) || resolved.from !== tdesignVueNextEsmEntry) {
        return resolved;
      }

      const componentName = String(resolved.name);
      const componentPath =
        tdesignComponentPathOverrides[componentName] ??
        componentName
          .replace(/^([A-Z]+)([A-Z][a-z])/u, '$1-$2')
          .replace(/([a-z0-9])([A-Z])/gu, '$1-$2')
          .toLowerCase();
      const componentStylePath = `${tdesignVueNextPackageName}/es/${componentPath}/style/css`;
      const componentStyleFile = path.resolve(
        CWD,
        'node_modules',
        tdesignVueNextPackageName,
        'es',
        componentPath,
        'style',
        'css.mjs',
      );

      return {
        ...resolved,
        from: `${tdesignVueNextPackageName}/es/${componentPath}`,
        ...(fs.existsSync(componentStyleFile) ? { sideEffects: componentStylePath } : {}),
      };
    },
  };
}

/**
 * 生成 Vue 项目的 Vite 配置。
 *
 * @param mode - 当前构建模式，用于控制自动导入、Mock 和代理相关配置
 * @returns Vite 用户配置对象
 */
export function createViteConfig(mode: string): UserConfig {
  const env = loadEnv(mode, CWD, '');
  const base = env.VITE_BASE_URL || '/';
  const apiPrefix = env.VITE_API_URL_PREFIX || '/api';
  const apiTarget = env.VITE_API_TARGET || 'http://127.0.0.1:3000';
  const proxyEnabled = env.VITE_IS_REQUEST_PROXY === 'true';
  const mockEnabled = mode === 'mock' || env.VITE_ENABLE_MOCK === 'true';
  const tdesignAutoImportEnabled = mode !== 'test';
  const developmentRoutesModule = path.resolve(
    CWD,
    mode === 'release' ? 'src/router/development-routes.release.ts' : 'src/router/development-routes.development.ts',
  );
  const rootHTTPProxyPaths = [
    '/healthz',
    '/docs',
    '/openapi.json',
    '/openapi.yaml',
    '/mcp/docs',
    '/mcp/docs.json',
    '/sse',
  ] as const;
  const websocketProxyPaths = ['/ws'] as const;

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
    // Monaco worker 必须走 Vite 原生 worker 转换，避免依赖预构建把 `?worker` 入口写成失效的优化 URL。
    optimizeDeps: {
      exclude: ['monaco-editor'],
    },
    build: {
      // Monaco 与 YAML worker 独立分包，编辑器代码按需加载，体积告警继续暴露依赖增长。
      chunkSizeWarningLimit: 4096,
      rolldownOptions: {
        output: {
          codeSplitting: {
            groups: [
              { name: 'vendor-monaco', test: /\/node_modules\/(?:monaco-editor|monaco-yaml)\//u },
              { name: 'vendor-tdesign-icons', test: /\/node_modules\/tdesign-icons-vue-next\//u },
              { name: 'vendor-tdesign', test: /\/node_modules\/(?:tdesign-vue-next|tvision-color)\//u },
              {
                name: 'vendor-vue',
                test: /\/node_modules\/(?:vue|@vue\/[^/]+|vue-router|pinia|vue-i18n|@vueuse\/core)\//u,
              },
              { name: 'vendor-utils', test: /\/node_modules\/lodash\//u },
            ],
          },
        },
      },
    },
    resolve: {
      alias: {
        '@/router/development-routes': developmentRoutesModule,
        '@': path.resolve(import.meta.dirname, './src'),
        'monaco-editor/esm/vs': monacoEditorEsmVsDirectory,
        'monaco-editor/min/vs/editor/editor.main.css': monacoEditorMainCssFile,
      },
    },
    css: {
      preprocessorOptions: {
        less: lessOptions,
      },
    },
    plugins: [
      ...(tdesignAutoImportEnabled
        ? [
            AutoImport({
              dts: false,
              resolvers: [createTDesignVueNextResolver()],
            }),
            Components({
              dts: false,
              resolvers: [createTDesignVueNextResolver()],
            }),
          ]
        : []),
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
              ws: true,
            } satisfies ProxyOptions,
            ...Object.fromEntries(
              websocketProxyPaths.map((proxyPath) => [
                proxyPath,
                {
                  target: apiTarget,
                  changeOrigin: true,
                  ws: true,
                } satisfies ProxyOptions,
              ]),
            ),
            ...Object.fromEntries(
              rootHTTPProxyPaths.map((proxyPath) => [
                proxyPath,
                {
                  target: apiTarget,
                  changeOrigin: true,
                } satisfies ProxyOptions,
              ]),
            ),
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
