// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
};

export default config;
