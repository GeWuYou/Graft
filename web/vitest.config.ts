import { mergeConfig } from 'vite';
import { defineConfig } from 'vitest/config';

import { createViteConfig } from './vite.config';

const tdesignComponentStylePattern = /tdesign-vue-next\/es\/.+\/style\/index\.css$/u;

export default mergeConfig(
  createViteConfig('test'),
  defineConfig({
    plugins: [
      {
        name: 'graft-vitest-tdesign-style-shim',
        resolveId(id) {
          if (tdesignComponentStylePattern.test(id)) {
            return '/src/test/empty-tdesign-style.css';
          }

          return undefined;
        },
      },
    ],
    test: {
      reporters: [
        'default',
        [
          '@d2t/vitest-ctrf-json-reporter',
          {
            outputDir: '.tmp/test-results',
            outputFile: 'ctrf-report.json',
          },
        ],
      ],
      passWithNoTests: true,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        reportsDirectory: './coverage',
      },
      css: true,
      server: {
        deps: {
          inline: [/tdesign-vue-next/u],
        },
      },
      environment: 'jsdom',
      exclude: ['ai-libs/**', 'coverage/**', 'dist/**', 'mock/**', 'node_modules/**'],
      setupFiles: ['./src/test/setup.ts'],
    },
  }),
);
