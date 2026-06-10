// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { I18nGovernanceRule } from '../types';
import { runLegacyRule } from './legacy-rule';
import { noDuplicateLocaleKeyRule } from './no-duplicate-locale-key';
import { noFallbackOnlyKeyFirstRule } from './no-fallback-only-key-first';
import { noHardcodedPluginMessageRule } from './no-hardcoded-plugin-message';
import { noHardcodedUiPropRule } from './no-hardcoded-ui-prop';
import { noLocaleCatalogDriftRule } from './no-locale-catalog-drift';
import { noMissingLocaleKeyRule } from './no-missing-locale-key';
import { noUnsafeDatetimeLocaleRule } from './no-unsafe-datetime-locale';
import { noUnsafeLocaleValueRule } from './no-unsafe-locale-value';
import { noUnusedLocaleKeyRule } from './no-unused-locale-key';

const legacyRule: I18nGovernanceRule = {
  id: 'legacy',
  description: 'Compatibility wrapper for remaining hard-coded UI text and system config schema fallback checks.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts', 'tsx', 'locale', 'go', 'schema'],
  check(context) {
    return runLegacyRule(context);
  },
};

export const rules: I18nGovernanceRule[] = [
  noMissingLocaleKeyRule,
  noLocaleCatalogDriftRule,
  noUnusedLocaleKeyRule,
  noDuplicateLocaleKeyRule,
  noUnsafeDatetimeLocaleRule,
  noUnsafeLocaleValueRule,
  noHardcodedUiPropRule,
  noHardcodedPluginMessageRule,
  noFallbackOnlyKeyFirstRule,
  legacyRule,
];
