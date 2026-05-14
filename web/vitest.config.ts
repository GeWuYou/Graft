import { mergeConfig } from 'vite';
import { defineConfig } from 'vitest/config';

import { createViteConfig } from './vite.config';

export default mergeConfig(
  createViteConfig('test'),
  defineConfig({
    test: {
      passWithNoTests: true,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        reportsDirectory: './coverage',
      },
      css: true,
      environment: 'jsdom',
      exclude: ['ai-libs/**', 'coverage/**', 'dist/**', 'mock/**', 'node_modules/**'],
      setupFiles: ['./src/test/setup.ts'],
    },
  }),
);
