export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', ['build', 'chore', 'ci', 'docs', 'feat', 'fix', 'perf', 'refactor', 'style', 'test']],
    'scope-empty': [2, 'never'],
    'subject-empty': [2, 'never'],
  },
};
