export default {
  defaultSeverity: 'error',
  extends: ['stylelint-config-standard'],
  ignoreFiles: ['ai-libs/**', 'coverage/**', 'dist/**', 'node_modules/**'],
  plugins: ['stylelint-order'],
  rules: {
    'custom-property-pattern': null,
    'declaration-property-value-no-unknown': null,
    'import-notation': 'string',
    'media-query-no-invalid': null,
    'no-descending-specificity': null,
    'no-empty-source': null,
    'order/properties-alphabetical-order': true,
    'selector-class-pattern': null,
    'selector-pseudo-class-no-unknown': [
      true,
      {
        ignorePseudoClasses: ['deep'],
      },
    ],
  },
  overrides: [
    {
      files: ['**/*.vue'],
      customSyntax: 'postcss-html',
    },
    {
      files: ['**/*.less'],
      customSyntax: 'postcss-less',
    },
  ],
};
