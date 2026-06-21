import { collectExactDuplicateKeys, collectLocaleCatalogs, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule } from '../types';

const SUGGESTION = 'Keep one exact key definition in each locale catalog file.';

export const noDuplicateLocaleKeyRule: I18nGovernanceRule = {
  id: 'no-duplicate-locale-key',
  description: 'Blocks duplicate exact locale keys inside a single locale catalog.',
  defaultSeverity: 'error',
  appliesTo: ['locale'],
  check(context) {
    return collectLocaleCatalogs(context).flatMap((catalog) =>
      collectExactDuplicateKeys(catalog).map((duplicate) =>
        localeViolation(
          noDuplicateLocaleKeyRule.id,
          'error',
          duplicate.file,
          `duplicate locale key ${duplicate.key} for ${catalog.locale}`,
          SUGGESTION,
          duplicate.line,
        ),
      ),
    );
  },
};
