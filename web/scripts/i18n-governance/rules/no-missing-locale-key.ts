import { collectLocaleCatalogs, collectRuntimeReferenceSet, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const SUGGESTION = 'Add the referenced key to the zh-CN/en-US locale catalogs owned by its boundary.';

export const noMissingLocaleKeyRule: I18nGovernanceRule = {
  id: 'no-missing-locale-key',
  description:
    'Blocks required frontend and server locale key references when no zh-CN/en-US locale catalog defines the key.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts', 'tsx', 'go', 'locale', 'schema'],
  check(context) {
    const catalogs = collectLocaleCatalogs(context);
    const referenceSet = collectRuntimeReferenceSet(context);
    const definedKeys = new Set<string>();
    const violations: RuleViolation[] = [];

    for (const catalog of catalogs) {
      for (const key of catalog.messages.keys()) definedKeys.add(key);
    }

    for (const key of [...referenceSet.requiredKeys].sort()) {
      if (!definedKeys.has(key)) {
        violations.push(
          localeViolation(
            noMissingLocaleKeyRule.id,
            'error',
            'src',
            `referenced locale key ${key} is missing from locale catalogs`,
            SUGGESTION,
          ),
        );
      }
    }

    return violations;
  },
};
