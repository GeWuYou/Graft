// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export default {
  defaultSeverity: 'error',
  extends: ['stylelint-config-standard'],
  ignoreFiles: ['ai-libs/**', 'coverage/**', 'dist/**', 'node_modules/**', 'mock/**'],
  plugins: ['stylelint-order'],
  rules: {
    'import-notation': 'string',
    'no-empty-source': null,
    'no-descending-specificity': null,
    'custom-property-pattern': null,
    'selector-class-pattern': null,
    'media-query-no-invalid': null,
    'declaration-property-value-no-unknown': null,
    'order/properties-alphabetical-order': true,
    'selector-pseudo-class-no-unknown': [
      true,
      {
        ignorePseudoClasses: ['deep'],
      },
    ],
  },
  overrides: [
    {
      files: ['**/*.html', '**/*.vue'],
      customSyntax: 'postcss-html',
    },
    {
      files: ['**/*.less'],
      customSyntax: 'postcss-less',
    },
  ],
};
