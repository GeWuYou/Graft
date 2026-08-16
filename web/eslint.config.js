import js from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import simpleImportSort from 'eslint-plugin-simple-import-sort';
import unusedImports from 'eslint-plugin-unused-imports';
import vue from 'eslint-plugin-vue';
import globals from 'globals';
import tseslint from 'typescript-eslint';
import vueParser from 'vue-eslint-parser';

export default tseslint.config(
  {
    ignores: [
      'ai-libs/**',
      'mock/**',
      'coverage/**',
      'dist/**',
      'node_modules/**',
      '.tmp/**',
      '_site/**',
      'temp*/**',
      '*.timestamp-*',
      'scripts/i18n-governance/fixtures/**',
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...vue.configs['flat/recommended'],
  {
    files: ['**/*.{js,mjs,cjs,ts,mts,cts,tsx,vue}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.node,
        ...globals.vitest,
        defineEmits: 'readonly',
        defineExpose: 'readonly',
        defineOptions: 'readonly',
        defineProps: 'readonly',
        withDefaults: 'readonly',
      },
    },
    plugins: {
      'simple-import-sort': simpleImportSort,
      'unused-imports': unusedImports,
    },
    rules: {
      eqeqeq: ['error', 'always'],
      'no-console': 'error',
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: 'consola',
              message: '业务代码只能通过 createLogger 使用日志，不要直接依赖 consola。',
            },
          ],
        },
      ],
      'no-debugger': 'error',
      'simple-import-sort/imports': 'error',
      'simple-import-sort/exports': 'error',
      'unused-imports/no-unused-imports': 'error',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
    },
  },
  {
    files: ['**/*.vue'],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        ecmaVersion: 'latest',
        extraFileExtensions: ['.vue'],
        parser: tseslint.parser,
        sourceType: 'module',
      },
    },
    rules: {
      'vue/block-lang': [
        'error',
        {
          script: {
            lang: ['ts'],
          },
        },
      ],
      'vue/block-order': [
        'error',
        {
          order: ['template', 'script', 'style'],
        },
      ],
      'vue/component-name-in-template-casing': ['error', 'kebab-case'],
      'vue/custom-event-name-casing': ['error', 'kebab-case'],
      'vue/multi-word-component-names': 'off',
      'vue/no-reserved-props': 'off',
      'vue/no-v-html': 'off',
      'vue/padding-line-between-blocks': ['error', 'never'],
      'vue/enforce-style-attribute': [
        'error',
        {
          allow: ['scoped'],
        },
      ],
    },
  },
  {
    files: ['src/test/**/*.ts', 'src/**/*.test.ts', 'src/**/*.test.tsx'],
    rules: {
      'vue/one-component-per-file': 'off',
      'vue/order-in-components': 'off',
      'vue/require-prop-types': 'off',
    },
  },
  {
    files: ['src/utils/logger/transports/**/*.ts'],
    rules: {
      'no-restricted-imports': 'off',
    },
  },
  {
    files: ['src/layouts/components/Header.vue'],
    rules: {
      'vue/enforce-style-attribute': ['error', { allow: ['plain', 'scoped'] }],
    },
  },
  eslintConfigPrettier,
);
