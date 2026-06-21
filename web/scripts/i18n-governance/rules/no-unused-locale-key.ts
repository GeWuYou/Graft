import {
  collectLocaleCatalogs,
  collectRuntimeReferenceSet,
  isAllowedUnusedLocaleKey,
  isRuntimeReferenced,
  localeViolation,
} from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const SUGGESTION = 'Remove unused locale keys or add a stable frontend/server reference for externally consumed keys.';

export const noUnusedLocaleKeyRule: I18nGovernanceRule = {
  id: 'no-unused-locale-key',
  description: 'Blocks locale catalog keys that are not referenced by frontend or server key consumers.',
  defaultSeverity: 'error',
  appliesTo: ['locale', 'vue', 'ts', 'tsx', 'go', 'schema'],
  check(context) {
    const catalogs = collectLocaleCatalogs(context);
    const referenceSet = collectRuntimeReferenceSet(context);
    const keyDefinitions = new Map<string, Set<string>>();
    const violations: RuleViolation[] = [];

    for (const catalog of catalogs) {
      for (const key of catalog.messages.keys()) {
        const files = keyDefinitions.get(key) ?? new Set<string>();
        files.add(catalog.file);
        keyDefinitions.set(key, files);
      }
    }

    for (const [key, files] of [...keyDefinitions.entries()].sort(([left], [right]) => left.localeCompare(right))) {
      if (isRuntimeReferenced(key, referenceSet) || isAllowedUnusedLocaleKey(key)) continue;

      violations.push(
        localeViolation(
          noUnusedLocaleKeyRule.id,
          'error',
          [...files].sort().join(', '),
          `unused locale key ${key}`,
          SUGGESTION,
        ),
      );
    }

    return violations;
  },
};
