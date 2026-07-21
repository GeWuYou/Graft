import type { KnipConfig } from 'knip';

const config: KnipConfig = {
  entry: ['src/**/*.vue'],
  project: [
    'src/**/*.{ts,tsx,js,jsx,vue}',
    // 测试文件不属于首轮运行面死代码治理范围，直接从扫描集合排除，避免留下配置提示噪音。
    '!src/**/*.test.*',
    '*.config.{ts,js,mjs,cjs}',
  ],
  ignoreDependencies: [
    // 提交校验配置由仓库根 hook/commit 流程间接使用，不能按前端运行面误删。
    '@commitlint/cli',
    // pre-commit 直接通过 node_modules/.bin/lint-staged 调用，Knip 无法从 shell hook 静态追踪到该依赖。
    'lint-staged',
  ],
  ignoreIssues: {
    // B2 先建立 shared 响应式公共 API；B3 的 Responsive primitives 首次消费后必须删除此豁免。
    'src/shared/composables/useContainerSize.ts': ['exports'],
    'src/shared/composables/useResponsiveVariant.ts': ['exports'],
    'src/shared/responsive/**': ['exports'],
  },
};

export default config;
