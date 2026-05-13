import { mergeConfig } from 'vite';
import { defineConfig } from 'vitest/config';

import viteConfig from './vite.config';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        reportsDirectory: './coverage',
      },
      css: true,
      environment: 'jsdom',
      exclude: ['ai-libs/**', 'coverage/**', 'dist/**', 'node_modules/**'],
      setupFiles: ['./src/test/setup.ts'],
    },
  }),
);
