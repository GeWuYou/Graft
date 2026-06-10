// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { collectLocaleCatalogs, localePairKey, localeViolation } from '../locale-utils';
import { normalizeText } from '../text-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const SELF_KEY_SUGGESTION = 'Replace placeholder locale values with real zh-CN/en-US display copy.';
const ENGLISH_CASE_SUGGESTION = 'Capitalize visible English UI copy unless the value is an allowed grammar fragment.';

function isEnglishInitialCaseExempt(key: string): boolean {
  const conjunctionKey = ['common', 'conjunction'].join('.');
  return key === conjunctionKey || key.endsWith('.unit');
}

function startsWithLowercaseLetter(value: string): boolean {
  return /^[a-z]/.test(normalizeText(value));
}

export const noUnsafeLocaleValueRule: I18nGovernanceRule = {
  id: 'no-unsafe-locale-value',
  description: 'Blocks locale values that resolve to their own key and warns on unsafe English initial casing.',
  defaultSeverity: 'error',
  appliesTo: ['locale'],
  check(context) {
    const violations: RuleViolation[] = [];

    for (const catalog of collectLocaleCatalogs(context)) {
      for (const [key, value] of catalog.messages) {
        if (normalizeText(value) === key) {
          violations.push(
            localeViolation(
              noUnsafeLocaleValueRule.id,
              'error',
              localePairKey(catalog.file),
              `locale key ${key} resolves to itself`,
              SELF_KEY_SUGGESTION,
            ),
          );
        }

        if (catalog.locale === 'en-US' && !isEnglishInitialCaseExempt(key) && startsWithLowercaseLetter(value)) {
          violations.push(
            localeViolation(
              noUnsafeLocaleValueRule.id,
              'warning',
              catalog.file,
              `English locale value for ${key} should start with an uppercase letter`,
              ENGLISH_CASE_SUGGESTION,
            ),
          );
        }
      }
    }

    return violations.sort((left, right) => {
      if (left.filePath !== right.filePath) return left.filePath.localeCompare(right.filePath);
      return left.message.localeCompare(right.message);
    });
  },
};
